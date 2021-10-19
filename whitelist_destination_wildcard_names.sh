#!/usr/bin/env bash

# whitelist_destination_wildcard_names.sh: whitelists a line delineated list of wildcarded dns objects as destinations
# Each line/entry in the $1 file should be the portion _after_, and _not_ including, "*.".  Example:
#	awsdns-41.org        would whitelist "*.awsdns-41.org"
# The filename should be the sole parameter to the script.  To read from stdin, use a filename of "-"

# To set a comment to use for all additions, run this before starting the script (without the leading "#"):
#	export WL_COMMENT="Put your comment here"

# version 0.0.3
# author: Logan Lembke (wildcard dns modifications by Bill S)


set -o noglob			#To avoid expanding the "*." in the next function

whitelist_wildcard_name_as_destination() {
    curl -XPOST "http://${HUNTER_INTERNAL_IP}:8080/api/v0/dnscat2-ja3/whitelist/add" --data-binary @- << EOF
{
    "name": "*.$1",
    "type": "domain_pattern",
    "comment": "$wl_comment",
    "domain": "$1"
}
EOF
}

HUNTER_INTERNAL_IP=`sudo docker inspect achunter_api -f "{{.NetworkSettings.Networks.aihunter_default.IPAddress}}" 2>/dev/null`
if [ -z "$HUNTER_INTERNAL_IP" ]; then
	HUNTER_INTERNAL_IP=`sudo docker inspect aihunter_api -f "{{.NetworkSettings.Networks.aihunter_default.IPAddress}}"`
fi
if [ -z "$HUNTER_INTERNAL_IP" ]; then
    echo "Unable to locate the API Container, exiting." >&2
    exit 1
fi

wl_comment="${WL_COMMENT:-Externally added entry}"


for domain in `cat $1`; do
    if [ -n "$domain" ]; then
        printf "Whitelisting $domain: "
        whitelist_wildcard_name_as_destination "$domain"
        printf "\n"
    fi
done
