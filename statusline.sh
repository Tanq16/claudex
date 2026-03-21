#!/bin/bash

# Colors
RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BLUE='\033[0;34m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

# Emojis
FIRE="🔥"
WARN="⚠️"

# Determine account label from this script's parent directory name
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_DIR_NAME="$(basename "$SCRIPT_DIR")"
if [[ "$CONFIG_DIR_NAME" == ".claude" ]]; then
    ACCT_LABEL="first"
elif [[ "$CONFIG_DIR_NAME" == ".claude2" ]]; then
    ACCT_LABEL="second"
elif [[ "$CONFIG_DIR_NAME" == ".claude3" ]]; then
    ACCT_LABEL="third"
else
    ACCT_NUM="${CONFIG_DIR_NAME##.claude}"
    ACCT_LABEL="acct${ACCT_NUM}"
fi

# Read JSON input from stdin (Claude Code pipes session data here)
input=$(cat)

# Extract values from Claude Code's JSON
cwd=$(echo "$input" | jq -r '.workspace.current_dir')
model_id=$(echo "$input" | jq -r '.model.id')
max_tokens=$(echo "$input" | jq -r '.context_window.context_window_size // empty')
session_used_pct=$(echo "$input" | jq -r '.context_window.used_percentage // empty')

# Replace home path with ~
cwd="${cwd/#$HOME/~}"

# Shorten directory if too long
if [[ ${#cwd} -gt 40 ]]; then
    cwd="...${cwd: -37}"
fi

# Convert model ID to shorthand
model_short=""
case "$model_id" in
  *"claude-opus-4"*)
    if [ "$max_tokens" = "1000000" ]; then
      model_short="opus[1m]"
    else
      model_short="opus"
    fi
    ;;
  *"claude-sonnet-4"*)
    if [ "$max_tokens" = "1000000" ]; then
      model_short="sonnet[1m]"
    else
      model_short="sonnet"
    fi
    ;;
  *"claude-3-7-sonnet"*|*"claude-3-5-sonnet"*)
    model_short="sonnet"
    ;;
  *"claude-3-5-haiku"*|*"claude-3-haiku"*)
    model_short="haiku"
    ;;
  *"claude-3-opus"*)
    model_short="opus"
    ;;
  *)
    model_short=$(echo "$input" | jq -r '.model.display_name')
    ;;
esac

# Colorize percentage
colorize_pct() {
    local val="$1"
    local show_emoji="${2:-false}"

    if (( $(echo "$val >= 90" | bc -l) )); then
        [[ "$show_emoji" == "true" ]] && echo -ne "${FIRE} "
        echo -ne "${RED}${val}%${NC}"
    elif (( $(echo "$val >= 70" | bc -l) )); then
        [[ "$show_emoji" == "true" ]] && echo -ne "${WARN} "
        echo -ne "${YELLOW}${val}%${NC}"
    else
        echo -ne "${GREEN}${val}%${NC}"
    fi
}

# Convert epoch timestamp to local time (e.g., "2pm")
epoch_to_time() {
    local epoch="$1"
    [[ -z "$epoch" ]] || [[ "$epoch" == "null" ]] || [[ "$epoch" == "0" ]] && return
    # Round up to next hour
    local rounded=$(( epoch + 3599 ))
    local formatted=$(date -j -f %s "$rounded" "+%-I%p" 2>/dev/null)
    echo "$formatted" | sed 's/AM/am/' | sed 's/PM/pm/'
}

# Read rate_limits directly from the stdin JSON — this is the current account's usage
five_h=""
seven_d=""
reset_time=""

five_h_raw=$(echo "$input" | jq -r '.rate_limits.five_hour.used_percentage // empty' 2>/dev/null)
seven_d_raw=$(echo "$input" | jq -r '.rate_limits.seven_day.used_percentage // empty' 2>/dev/null)
five_h_reset=$(echo "$input" | jq -r '.rate_limits.five_hour.resets_at // empty' 2>/dev/null)

if [[ -n "$five_h_raw" ]]; then
    five_h=$(printf "%.0f" "$five_h_raw" 2>/dev/null || echo "0")
fi
if [[ -n "$seven_d_raw" ]]; then
    seven_d=$(printf "%.0f" "$seven_d_raw" 2>/dev/null || echo "0")
fi
if [[ -n "$five_h_reset" ]]; then
    reset_time=$(epoch_to_time "$five_h_reset")
fi

# Get git branch
original_cwd=$(echo "$input" | jq -r '.workspace.current_dir')
git_branch=""
if git -C "$original_cwd" rev-parse --git-dir > /dev/null 2>&1; then
    git_branch=$(git -C "$original_cwd" branch --show-current 2>/dev/null)
fi

# Build status line: account | model | directory | branch | context info
out="${BOLD}${YELLOW}${ACCT_LABEL}${NC}"
out+=" | ${CYAN}${BOLD}${model_short}${NC}"
out+=" | ${cwd}"

if [[ -n "$git_branch" ]]; then
    out+=" | ${MAGENTA}${git_branch}${NC}"
fi

if [[ -n "$session_used_pct" ]]; then
    sess_pct=$(printf "%.0f" "$session_used_pct")
    out+=" | ${BLUE}ctx: ${sess_pct}%${NC}"
fi

if [[ -n "$five_h" ]]; then
    out+=" | 5h: $(colorize_pct "$five_h" "true")"
    if [[ -n "$reset_time" ]]; then
        out+=" ${DIM}(${reset_time})${NC}"
    fi
fi

if [[ -n "$seven_d" ]]; then
    out+=" | Wk: $(colorize_pct "$seven_d" "false")"
fi

# Output
echo -e "$out"
