#!/usr/bin/env bash

STATUS_FILE="$HOME/.config/z103_eq_status.txt"
mkdir -p "$(dirname "$STATUS_FILE")"

FREQS=(25 40 63 100 160 250 400 630 1000 1600 2500 4000 6300 10000 16000)
LABELS=("25 Hz" "40 Hz" "63 Hz" "100 Hz" "160 Hz" "250 Hz" "400 Hz" "630 Hz" "1 kHz" "1.6 kHz" "2.5 kHz" "4 kHz" "6.3 kHz" "10 kHz" "16 kHz")
NUM_BANDS=${#FREQS[@]}

BAR_MAX="##############################" 
BG_MAX="------------------------------"  

if [ ! -f "$STATUS_FILE" ]; then
    GAINS=(0 0 0 -8 -6 0 0 0 0 0 0 0 0 0 0)
    echo "${GAINS[*]}" > "$STATUS_FILE"
else
    read -r -a GAINS < "$STATUS_FILE"
    if [ ${#GAINS[@]} -ne $NUM_BANDS ]; then
        GAINS=(0 0 0 -8 -6 0 0 0 0 0 0 0 0 0 0)
    fi
fi

CURSOR=3       

save_to_disk() {
    echo "${GAINS[*]}" > "$STATUS_FILE"
}

check_routing() {
    if pactl list modules short 2>/dev/null | grep -q "sink_name=z103_eq"; then
        return 0
    fi
    return 1
}

# MANUAL ACTION: HIT SPACEBAR TO COMMIT GAINS LIVE
apply_gains() {
    save_to_disk
    local i
    for i in "${!FREQS[@]}"; do
        pw-cli set-param z103_eq Props "{ controlOutputs = [ { node = eq$i property = Gain value = ${GAINS[$i]}.0 } ] }" >/dev/null 2>&1
    done
}

# MANUAL ACTION: PRESS "M" TO TURN ON THE EQ DEVICE
enable_routing() {
    save_to_disk
    
    # Clean up old modules first
    local old_mod=$(pactl list modules short 2>/dev/null | grep "sink_name=z103_eq" | awk '{print $1}')
    if [ -n "$old_mod" ]; then
        pactl unload-module "$old_mod" >/dev/null 2>&1
    fi
    sleep 0.1

    local NODES_BLOCK=""
    local i
    for i in "${!FREQS[@]}"; do
        NODES_BLOCK+="{ type = builtin name = eq$i label = bq_peaking control = { Freq = ${FREQS[$i]}.0 Q = 1.0 Gain = ${GAINS[$i]}.0 } } "
    done

    local LINKS_BLOCK=""
    for ((i=0; i<NUM_BANDS-1; i++)); do
        local next=$((i+1))
        LINKS_BLOCK+="{ output = \"eq$i:Out\" input = \"eq$next:In\" } "
    done

    # Load via native module-filter-chain wrapper (creates a direct system hardware sink)
    pactl load-module libpipewire-module-filter-chain args="{ node.description=\"Z103 Equalizer\" media.name=\"Z103 Equalizer\" filter.graph={ nodes=[ $NODES_BLOCK ] links=[ $LINKS_BLOCK ] } capture.props={ sink_name=z103_eq node.name=z103_eq node.description=\"Z103 Equalizer\" media.class=Audio/Sink audio.position=[FL FR] } playback.props={ node.passive=true audio.position=[FL FR] } }" >/dev/null 2>&1

    sleep 0.2
    
    # Enforce routing across the operating system mixer
    pactl set-default-sink z103_eq >/dev/null 2>&1
    local inputs=$(pactl list sink-inputs short 2>/dev/null | awk '{print $1}')
    for input in $inputs; do
        pactl move-sink-input "$input" z103_eq >/dev/null 2>&1
    done
}

# MANUAL ACTION: PRESS "M" TO TURN OFF THE EQ DEVICE
disable_routing() {
    local mod_id=$(pactl list modules short 2>/dev/null | grep "sink_name=z103_eq" | awk '{print $1}')
    if [ -n "$mod_id" ]; then
        pactl unload-module "$mod_id" >/dev/null 2>&1
    fi
    
    sleep 0.1
    local fallback=$(pactl list sinks short 2>/dev/null | grep -v "z103_eq" | head -n 1 | awk '{print $2}')
    if [ -n "$fallback" ]; then
        pactl set-default-sink "$fallback" >/dev/null 2>&1
        local inputs=$(pactl list sink-inputs short 2>/dev/null | awk '{print $1}')
        for input in $inputs; do
            pactl move-sink-input "$input" "$fallback" >/dev/null 2>&1
        done
    fi
}

draw_ui() {
    clear
    local GREEN=$(tput setaf 2)
    local RED=$(tput setaf 1)
    local YELLOW=$(tput setaf 3)
    local BLUE=$(tput setaf 4)
    local RESET=$(tput sgr0)
    local BOLD=$(tput bold)

    echo "${BLUE}=========================================================================${RESET}"
    echo "${BOLD}               MANUAL INTEGRATED 15-BAND AUDIO CONTROLLER                ${RESET}"
    echo "${BLUE}=========================================================================${RESET}"
    echo ""

    check_routing
    if [ $? -eq 0 ]; then
        echo "  Routing Status: ${GREEN}${BOLD}[ ACTIVE -> ROUTED THROUGH EQ ]${RESET}"
    else
        echo "  Routing Status: ${RED}${BOLD}[ BYPASS -> DIRECT HARDWARE SOUND ]${RESET}"
    fi
    echo ""

    local i
    for i in "${!FREQS[@]}"; do
        local current_gain=${GAINS[$i]}
        local selector="  "
        if [ "$CURSOR" -eq "$i" ]; then
            selector="${BLUE}->${RESET}"
        fi

        local filled_blocks=$((current_gain + 15))
        local empty_blocks=$((30 - filled_blocks))
        local visual_bar="${BAR_MAX:0:filled_blocks}${BG_MAX:0:empty_blocks}"

        printf "%b %-9s [%s] %s%3d dB%b\n" "$selector" "${LABELS[$i]}" "$visual_bar" "${YELLOW}" "$current_gain" "${RESET}"
    done

    echo ""
    echo "${BLUE}-------------------------------------------------------------------------${RESET}"
    echo "  ${BOLD}Controls:${RESET}"
    echo "  [↑ / ↓] or (W / S) : Move Slider        [← / →] or (A / D) : Change Gain"
    echo "  [0] Reset Active Band"
    echo "  ${BOLD}[SPACE]            : Commit & Apply Current Gain Profile Live${RESET}"
    echo "  ${BOLD}[M]                : Toggle On/Off (Switch between EQ and Raw Hardware)${RESET}"
    echo "  [Q] Quit Interface"
    echo "${BLUE}-------------------------------------------------------------------------${RESET}"
    echo -n "  Tune > "
}

draw_ui

while true; do
    read -s -n 1 key
    
    if [[ "$key" == $'\e' ]]; then
        read -s -n 2 -t 0.02 nest_key
        key+="$nest_key"
    fi
    
    case "$key" in
        $'\e[A'|[wW])
            if [ "$CURSOR" -gt 0 ]; then ((CURSOR--)); fi
            ;;
        $'\e[B'|[sS])
            if [ "$CURSOR" -lt $((NUM_BANDS - 1)) ]; then ((CURSOR++)); fi
            ;;
        $'\e[D'|[aA])
            if [ "${GAINS[$CURSOR]}" -gt -15 ]; then
                ((GAINS[$CURSOR]--))
                save_to_disk  
            fi
            ;;
        $'\e[C'|[dD])
            if [ "${GAINS[$CURSOR]}" -lt 15 ]; then
                ((GAINS[$CURSOR]++))
                save_to_disk  
            fi
            ;;
        0)
            GAINS[$CURSOR]=0
            save_to_disk      
            ;;
        " ")
            apply_gains
            ;;
        mM)
            check_routing
            if [ $? -eq 0 ]; then
                disable_routing
            else
                enable_routing
            fi
            ;;
        q|Q)
            clear
            exit 0
            ;;
    esac

    draw_ui
done
