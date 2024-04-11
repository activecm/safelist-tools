#!/bin/bash

#Recommended call:
#	./download-safelist.sh
#will download to safelist.yyyymmddhhmmss.json in the current directory

Now=`date '+%Y%m%d%H%M%S'`

#We need to have jq installed to strip out the _id field from the safelist
type -path jq >/dev/null 2>&1 || sudo yum -y install jq || sudo apt -y install jq

#Make sure we have the dnscat2-ja3 database still loaded as we use that database in the API call
rita list 2>/dev/null | grep -q dnscat2-ja3 2>&1 || /opt/AC-Hunter/scripts/init_mongo_data.sh


#Make a call to the AC-Hunter API, requesting the whitelist for dnscat2-ja3.  Remove the _id field from each whitelist entry as that's internal and not useful.
curl --no-progress-meter -XGET "http://$(sudo docker inspect achunter_api -f '{{.NetworkSettings.Networks.achunter_default.IPAddress}}'):8080/api/v0/dnscat2-ja3/whitelist" \
 | jq 'map(with_entries(select(.key != "_id")))' \
 >"safelist.$Now.json"

echo "Your safelist is stored in the file safelist.$Now.json" >&2
