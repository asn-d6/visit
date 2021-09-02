import sqlite3
import os
import json
import math

VALS_PER_JSON = 1000

def get_validators(db, range_start, range_end):
    """Get validators (ordered by validator index) from 'range_start' to 'range_end'"""
    rows = db.execute("""SELECT * from validator_state ORDER BY validator_idx LIMIT %d OFFSET %d"""
                      % (range_end, range_start)).fetchall()

    return json.dumps( [dict(ix) for ix in rows] ) #CREATE JSON


db_fname = "../foo.db"
conn = sqlite3.connect( db_fname )
conn.row_factory = sqlite3.Row # This enables column access by name: row['column_name'] 
db = conn.cursor()

# Extract some metadata from the db
db.execute("SELECT COUNT(DISTINCT validator_idx) FROM validator_state")
n_validators = db.fetchone()[0]
db.execute("SELECT COUNT(DISTINCT epoch) FROM validator_state")
epochs = db.fetchone()[0]
db.execute("SELECT COUNT(*) FROM validator_state")
total_entries = db.fetchone()[0]
assert(total_entries == epochs * n_validators) # sanity check

n_jsons_needed = math.ceil(n_validators / VALS_PER_JSON)
print("For %d validators and %d per json, we need %d jsons (%d total entries)" %
      (n_validators, VALS_PER_JSON, n_jsons_needed, total_entries))

# Each validator is present in multiple epochs, so entries per json depend on the number of validators and number of epochs
entries_per_json = VALS_PER_JSON*epochs

i = 0
if not os.path.exists('./data'):
    os.mkdir("./data")
for cur in range(0, total_entries, entries_per_json):
    json_file = open("data/data%d.json" % (i), "w")
    print("Writing entries %d to %d" % (cur, cur+entries_per_json))
    # Print something that javascript will understand
    json_str = "var data = %s\n" % (get_validators(db, cur, cur+entries_per_json))
    json_file.write(json_str)
    json_file.close()
    i+=1

conn.commit()
conn.close()

