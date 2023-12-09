# Host Persister

This allows for hosts specified in a host formatted file to be synced to the actual hosts file so-long as their entries do not already exist.

The .env file is used to configure the hosts file to use a source (SOURCE_FILE) to be synced to the actual file (HOSTS_FILE).
There is the ability to overwrite existing values as part of sync, so long as the domain is the only one defined on the line (HOSTS_OVERWRITE = 1).
There is the ability to periodic sync with a specified interval (SYNC_TIME).

Maintainer: 
[Alfred Manville](https://code.mrmelon54.com/alfred)

License: 
BSD 3-Clause