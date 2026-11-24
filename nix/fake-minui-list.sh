while [ $# -gt 0 ]; do
    case $1 in
    --file) file=$2; shift ;;
    --write-location) output=$2; shift ;;
    --format | --item-key | --title | --title-alignment | --write-value | --confirm-text | --cancel-text | --action-text | --background-color | --background-image) shift ;;
    --disable-auto-sleep | --hide-hardware-group) ;;
    *)
        echo "minui-list: invalid option -- '$1'" >&2
        exit 1
        ;;
    esac
    shift
done

jq -c '[.items[] | if .options then .name + "=" + .options[.selected // 0] else .name end]' "$file" >>/tmp/menus.log

choice=$(head -n 1 /tmp/choices 2>/dev/null || true)
if [ -z "$choice" ]; then
    exit 2
fi
sed -i 1d /tmp/choices

jq -r --arg prefix "$choice" '[.items[].name | select(startswith($prefix))][0]' "$file" >"$output"
