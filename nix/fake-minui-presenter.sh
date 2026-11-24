timeout=0
while [ $# -gt 0 ]; do
    case $1 in
    --message) message=$2; shift ;;
    --timeout) timeout=$2; shift ;;
    --confirm-text | --cancel-text | --action-text | --confirm-button | --cancel-button | --action-button | --message-alignment | --background-color | --background-image) shift ;;
    --confirm-show | --cancel-show | --action-show | --show-pill | --show-hardware-group | --show-time-left | --disable-auto-sleep) ;;
    *)
        echo "minui-presenter: invalid option -- '$1'" >&2
        exit 1
        ;;
    esac
    shift
done

printf '%s\n---\n' "$message" >>/tmp/messages.log

if [ "$timeout" -lt 0 ]; then
    exec sleep infinity
fi
