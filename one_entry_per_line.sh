#!/bin/bash

# version 0.0.2

#Turn the json structure (assumed to be a list at the highest level)
#into a one-line-per-array-entry equivalent json structure.

( echo '[' ; jq -c '.[]' | sed -e 's/^/    /' -e '$!s/$/,/' ; echo ']' )


