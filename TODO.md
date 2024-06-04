- the correct UI is actually:
    - main page, description, example visualisation, and LIST OF CACHED REPOS, sorted by stars or size or whatever
    - you can add a repo to the list and you're told that it will appear on the list in some minutes.
    
- clicking submit while a thing is running should kill it
- parsing the log info is stupid rn, should set command stdout to a buffer and read it as it's being written to until it's done. maybe even print progress on how many files have been parsed. 