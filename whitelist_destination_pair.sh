#!/usr/bin/env bash

# whitelist_destination_cidr.sh: whitelists a line delineated list of IPv4 address pairs (space in between) as destinations
# The filename should be the sole parameter to the script.  To read from stdin, use a filename of "-"

# To set a comment to use for all additions, run this before starting the script (without the leading "#"):
#	export WL_COMMENT="Put your comment here"

# version 0.0.4
# author: Logan Lembke (Minor changes by Bill S)

whitelist_pair_as_destination() {
    echo curl -XPOST "http://${HUNTER_INTERNAL_IP}:8080/api/v0/dnscat2-ja3/whitelist/add" --data-binary @- >&2
    echo { "name": "IP Pair: $1 -- $2", "type": "pair", "comment": "$wl_comment", "pair": { "dst": "$2", "dst_network_uuid": "/////////////////////w==", "src": "$1", "src_network_uuid": "/////////////////////g==" } } EOF >&2

    curl -XPOST "http://${HUNTER_INTERNAL_IP}:8080/api/v0/dnscat2-ja3/whitelist/add" --data-binary @- << EOF
{
  "comment": "$wl_comment",
  "name": "IP Pair: $1 -- $2",
  "pair": {
    "dst": "$2",
    "dst_network_uuid": "/////////////////////w==",
    "src": "$1",
    "src_network_uuid": "/////////////////////g=="
  },
  "type": "pair"
}
EOF
}


HUNTER_INTERNAL_IP=`sudo docker inspect achunter_api -f "{{.NetworkSettings.Networks.achunter_default.IPAddress}}" 2>/dev/null`
if [ -z "$HUNTER_INTERNAL_IP" ]; then
	HUNTER_INTERNAL_IP=`sudo docker inspect achunter_api -f "{{.NetworkSettings.Networks.aihunter_default.IPAddress}}" 2>/dev/null`
fi
if [ -z "$HUNTER_INTERNAL_IP" ]; then
	HUNTER_INTERNAL_IP=`sudo docker inspect aihunter_api -f "{{.NetworkSettings.Networks.aihunter_default.IPAddress}}"`
fi
if [ -z "$HUNTER_INTERNAL_IP" ]; then
    echo "Unable to locate the API Container, exiting." >&2
    exit 1
fi

wl_comment="${WL_COMMENT:-Externally added entry}"

cat $1 \
 | while read left_ip right_ip ; do
    if [ -n "$left_ip" -a -n "$right_ip" ]; then
        printf "Whitelisting pair $left_ip -- $right_ip: "
        whitelist_pair_as_destination "$left_ip" "$right_ip"
        printf "\n"
    fi
done
