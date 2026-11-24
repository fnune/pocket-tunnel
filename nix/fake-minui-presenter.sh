while [ $# -gt 0 ]; do
    case $1 in
    --message) printf '%s\n---\n' "$2" >>/tmp/messages.log; shift ;;
    esac
    shift
done
