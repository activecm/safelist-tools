#!/usr/bin/env python3
"""Add the hash_key field to entries in a whitelist json file."""

#Copyright 2021, William Stearns <bill@activecountermeasures.com>

import os			#For loading and saving raw files
import sys
import json			#To load and save json files




def load_json_from_file(json_filename):
	"""Bring in json content from a file and return it as a python data structure (or None if not successful for any reason)."""

	ljff_return = None

	if os.path.exists(json_filename) and os.access(json_filename, os.R_OK):
		try:
			with open(json_filename) as json_h:
				ljff_return = json.loads(json_h.read())
		except:
			pass

	return ljff_return


def write_object(filename, generic_object):
	"""Write out an object to a file."""

	try:
		with open(filename, "wb") as write_h:
			write_h.write(generic_object.encode('utf-8'))
	except:
		sys.stderr.write("Problem writing " + filename + ", skipping.")
		raise

	#return


def add_hashes_to_whitelist(orig_structure):
	"""Take a whitelist structure and for each entry, check that it has a hash_key and add it if not."""

	#print(str(orig_structure))

	new_structure = []

	for one_white_entry in orig_structure:
		if 'Type' in one_white_entry:
			print('This appears to be a previous whitelist format.  Please contact support@activecountermeasures.com .')
			sys.exit(1)

		if 'hash_key' not in one_white_entry:
			if one_white_entry['type'] == 'asn':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'asn_org':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'cidr':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'domain':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'domain_literal':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'domain_pattern':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'ip':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'org':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'pair':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'ranges':
				one_white_entry['hash_key'] = -99999
			elif one_white_entry['type'] == 'useragent':
				one_white_entry['hash_key'] = -99999
			else:
				print('Unrecognized type field.  Please contact support@activecountermeasures.com .')
				sys.exit(1)
		#else:
		#	#Consider comparing existing hash_key value to recalculated value, error and exit if different.


		new_structure.append(one_white_entry)

	return new_structure



add_hash_key_version = '0.0.2'


if __name__ == '__main__':

	orig_whitelist = load_json_from_file("./whitelist.json")
	hashed_whitelist = add_hashes_to_whitelist(orig_whitelist)
	#try:
	write_object("./whitelist-hashed.json", json.dumps(hashed_whitelist))
	#except:
	#	raise
