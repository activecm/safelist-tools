#!/usr/bin/env python3
"""Synchronize the whitelist on 2 or more AC-Hunter servers."""

__version__ = '0.0.4'

__author__ = 'William Stearns'
__copyright__ = 'Copyright 2021, William Stearns'
__credits__ = ['William Stearns']
__email__ = 'bill@activecountermeasures.com'
__license__ = 'GPL 3.0'
__maintainer__ = 'William Stearns'
__status__ = 'Prototype'                                #Prototype, Development or Production

#Copyright 2021, William Stearns <bill@activecountermeasures.com>
#Released under the GPL 3.0

import os
import sys
import time						#For sleeping
import json						#For converting between json and python lists
import errno
import requests						#For API calls


def debug_out(output_string):
	"""Send debugging output to stderr."""

	if cl_args['devel']:
		sys.stderr.write(output_string + '\n')
		sys.stderr.flush()


def fail(fail_string):
	"""Send failure message to stderr and exit."""

	if cl_args['devel']:
		sys.stderr.write(fail_string + ', exiting.\n')
		sys.stderr.flush()
	sys.exit(1)


def load_json_from_file(json_filename):
	"""Bring in json content from a file and return it as a python data structure (or None if not successful for any reason)."""

	ljff_return = None

	if os.path.exists(json_filename) and os.access(json_filename, os.R_OK):
		try:
			with open(json_filename) as json_h:
				ljff_return = json.loads(json_h.read())
		except:											# pylint: disable=bare-except
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


def mkdir_p(path):
	"""Create an entire directory branch.  Will not complain if the directory already exists."""

	if not os.path.isdir(path):
		try:
			os.makedirs(path)
		except FileExistsError:
			pass
		except OSError as exc:
			if exc.errno == errno.EEXIST and os.path.isdir(path):
				pass
			else:
				raise


def cache_file(parent_cache_dir, host_string):
	"""Returns the correct filename that would hold the whitelist for that host string.  Does not care if the file exists or not, but does create the directory that would hold it."""

	cache_obj_path = parent_cache_dir + '/'

	mkdir_p(cache_obj_path)

	return cache_obj_path + host_string + '.whitelist.json'


def api_call(target_url, verb, data_block):
	"""Place an API call."""

	api_response = None

	debug_out('API Request: ' + str(verb) + ' ' + str(target_url))
	try:
		if verb == 'GET':
			api_response = requests.get(target_url, timeout=30)
		elif verb == 'POST':
			api_response = requests.post(target_url, timeout=30, headers={'Content-Type': 'application/json'}, data=data_block)
		else:
			fail('Unhandled verb: ' + str(verb))
	except requests.exceptions.InvalidSchema:
		debug_out('InvalidSchema retrieving ' + str(target_url))
	#From urllib3
	#except LocationValueError:
	#	debug_out('LocationValueError retrieving ' + str(target_url))
	except requests.exceptions.InvalidURL:
		debug_out('Invalid label retrieving ' + str(target_url))
	except requests.exceptions.ReadTimeout:
		debug_out('Timeout retrieving ' + str(target_url))
	except requests.exceptions.TooManyRedirects:
		debug_out('Too many redirects retrieving ' + str(target_url))
	except requests.exceptions.SSLError:
		debug_out('SSL certificate error retrieving ' + str(target_url))
	except requests.exceptions.ConnectionError:
		debug_out('Connection error retrieving ' + str(target_url))
	except UnicodeError:
		debug_out('Unicode error retrieving ' + str(target_url))
	except requests.exceptions.ContentDecodingError:
		debug_out('Decoding/gzip error retrieving ' + str(target_url))

	return api_response


def get_whitelists(host_list):
	"""Pull down the whitelists from the listed hosts, return them in a dictionary (key=host, value=whitelist structure (list of whitelist dictionaries)."""

	host_whitelists = {}

	for one_host in host_list:
		if one_host:
			api_ret = api_call('http://' + str(one_host) + '/api/v0/empire/whitelist/export', 'GET', None)
			if api_ret:
				if api_ret.status_code == 200:
					host_whitelist = api_ret.json()					#Don't need json.loads() around it, it autoconverts to a python list.
					if host_whitelist:
						debug_out('Whitelist for ' + str(one_host) + ' has ' + str(len(host_whitelist)) + ' entries.')
						host_whitelists[one_host] = host_whitelist
					else:
						debug_out('Empty host whitelist for ' + str(one_host))
				else:
					debug_out('Status code for ' + str(one_host) + ' is ' + str(api_ret.status_code))
			elif api_ret is None:
				debug_out(str(one_host) + ' returned None ')

	return host_whitelists


def filter_by_comment(whitelist_tree, comment_filter):
	"""From a given whitelist tree, pull out _just_ the ones that have the comment_filter in the 'comment' value."""

	if not comment_filter:										# pylint: disable=no-else-return
		#comment_filter is empty, so no filtering requested.  Return the original structure.
		return whitelist_tree
	else:
		#There is a requested comment filter, so we need to filter by comment and return the filtered tree.
		lowercase_comment_filter = comment_filter.lower()

		filtered_tree = {}

		for one_host in whitelist_tree:
			filtered_tree[one_host] = []
			for one_white_entry in whitelist_tree[one_host]:
				if 'comment' in one_white_entry and one_white_entry['comment'] is not None and one_white_entry['comment'].lower().find(lowercase_comment_filter) != -1:		#Comment filter string found
					filtered_tree[one_host].append(one_white_entry)
			debug_out('(Only ' + str(len(filtered_tree[one_host])) + ' entries remain for ' + str(one_host) + ' after filtering.)')

		#debug_out(str(filtered_tree))

		return filtered_tree



def merge_whitelist_entries(dict_of_whitelists):
	"""Take all the whitelist entries from a tree and merge into a deduplicated list."""

	merged_whitelist = []

	for one_host in dict_of_whitelists:
		for one_white in dict_of_whitelists[one_host]:
			if one_white not in merged_whitelist:
				merged_whitelist.append(one_white)

	return merged_whitelist


def gen_host_additions(combined_whitelist, this_host_whitelist):
	"""Find a list of entries from combined that are not in this_host."""

	host_changes = []

	for one_white in combined_whitelist:
		if one_white not in this_host_whitelist and one_white not in host_changes:
			host_changes.append(one_white)

	return host_changes


def push_changes(target_host, entries_to_add):
	"""Send these changes back to the specific host."""

	debug_out('Intend to push ' + str(len(entries_to_add)) + ' entries to host ' + str(target_host))

	api_ret = api_call('http://' + str(target_host) + '/api/v0/empire/whitelist/import', 'POST', json.dumps(entries_to_add))
	if api_ret:
		if api_ret.status_code == 201:
			debug_out('Import appeared successful.')
		else:
			debug_out('Status code for ' + str(target_host) + ' is ' + str(api_ret.status_code))
		debug_out(str(api_ret.text))
	elif api_ret is None:
		debug_out(str(target_host) + ' returned None ')
	else:
		debug_out('Unknown return from post.')


def cache_whitelists(top_dir, whitelist_dict):
	"""Loop through each of the whitelists in the dict and save them under the host ID on disk."""

	for one_host in whitelist_dict:
		if whitelist_dict[one_host]:			#Don't write out an empty dictionary - this may mean we weren't able to retrieve it.
			debug_out('Writing cache file for ' + str(one_host))
			write_object(cache_file(top_dir, one_host), json.dumps(whitelist_dict[one_host]))


def process_whitelist_adds(master_whitelist, whitelist_dict):
	"""For each host, individually find a list of whitelist entries that need to be added and add them."""

	for one_ac_host in whitelist_dict:
		host_additions = gen_host_additions(master_whitelist, whitelist_dict[one_ac_host])
		if host_additions:
			debug_out(str(len(host_additions)) + ' unique changes to send to ' + str(one_ac_host))
			if cl_args['dryrun']:
				debug_out('Changes will not be sent (dryrun mode)')
			else:
				push_changes(one_ac_host, host_additions)
		else:
			debug_out('No changes needed for host ' + str(one_ac_host))



whitelist_cache_dir = os.environ["HOME"] + '/.cache/safelist-sync/'



if __name__ == '__main__':
	import argparse

	parser = argparse.ArgumentParser(description='safelist-sync version ' + str(__version__))
	parser.add_argument('-s', '--sources', help='System(s) that both provide and receive whitelist entries. Should be host:port', required=False, default=[], nargs='*')
	parser.add_argument('-r', '--recipients', help='System(s) that only receive whitelist entries', required=False, default=[], nargs='*')
	parser.add_argument('-f', '--filter', help='Text that must be in the comment field (case insensitive), otherwise that whitelist entry is ignored', required=False, default='')
	parser.add_argument('-w', '--wait', help='Seconds between checks (default: %(default)s)', type=int, required=False, default=300)
	parser.add_argument('-d', '--dryrun', help='Do not make any changes', required=False, default=False, action='store_true')
	parser.add_argument('--devel', help='Enable development/debug statements', required=False, default=False, action='store_true')
	(parsed, unparsed) = parser.parse_known_args()
	cl_args = vars(parsed)

	if len(cl_args['sources']) == 0:
		fail('No sources specified')
	if len(cl_args['sources']) + len(cl_args['recipients']) < 2:
		fail('Not enough systems to sync')
	#if any hosts in sources are also in recipients, delete from recipients and give a gentle warning.

	continue_loop = True

	while continue_loop:
		debug_out('Starting sync')
		raw_source_whitelists = get_whitelists(cl_args['sources'])
		source_whitelists = filter_by_comment(raw_source_whitelists, cl_args['filter'])
		raw_recipient_whitelists = get_whitelists(cl_args['recipients'])
		recipient_whitelists = filter_by_comment(raw_recipient_whitelists, cl_args['filter'])

		master_white_dict = merge_whitelist_entries(source_whitelists)

		process_whitelist_adds(master_white_dict, source_whitelists)
		process_whitelist_adds(master_white_dict, recipient_whitelists)

		cache_whitelists(whitelist_cache_dir, raw_source_whitelists)
		cache_whitelists(whitelist_cache_dir, raw_recipient_whitelists)

		debug_out('')
		try:
			time.sleep(cl_args['wait'])
		except KeyboardInterrupt:
			continue_loop = False
			debug_out('\nCtrl-C detected, exiting.')
