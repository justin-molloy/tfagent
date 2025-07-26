# tfagent

## File transfer agent for windows

![Build Status](https://github.com/justin-molloy/tfagent/actions/workflows/builder.yaml/badge.svg)

### Summary
The tfagent is a utility that monitors directories for files, and when a new file is created or updated in the watched directories, it transfers the file to a specified remote (sftp) location. The config also has some options for archiving the file to a secondary directory for verification and retransmission if necessary.

This project was borne out of the necessity for some legacy applications to transfer files out of a secure environment as part of their integration workflow - while there may be better ways of doing this, almost all others that I can thnk of involve either allowing external services to reach in to our secure environment, or some sort of transformation of the file before sending.

The intent for this app is that it's a single binary that can be copied to windows servers/workstations with minimal effort, and can be configured easily by the YAML config file, so slightly less technical people have no problem running it. In the future, a simple web interface might be added, but even that might be adding too much complexity.

The general workflow for this app is this:

### Workflow
1. Read config file for transfer definitions that show which directories to monitor
2. Set up a watcher function for the directories
3. Any changes to the directories are processed using parameters defined in the config, and eligible files are queued for transfer
4. A separate goroutine executes the file transfer and reports status.
5. Optional archiving of the file once the transfer is complete(not implemented yet).

### Config

Sample config file is below.

```
# Global settings

logdest: c:\data\code\testfolder\logs\logfile.log
loglevel: debug
delay: 1

# Individual transfer definitions
transfers:
  - name: Result Files 
    source_directory: c:\data\code\testfolder\tf1
    transfertype: sftp
    filter:
    username: tfuser
    privatekey: c:\data\code\testfolder\id_ed25519_test.priv
    password: abcd1234
    server: 192.168.214.128
    port: 22
    remotepath: incoming
    streaming: false
    overwrite_dest: false
    action_on_success: archive
    archive_dest: c:\data\code\testfolder\tf1\archive
    filter: "\\.(jpg|png|gif)$"

  - name: Log Files
    source_directory: c:\data\code\testfolder\tf2
    destination: /path/to/destination2
    streaming: True
```
