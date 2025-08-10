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
5. Optional archiving of the file once the transfer is complete.

### Build

To build the app into a transportable exe, use the go build command in the base working directory:
```
go build -o tfagent.exe tfagent.go
```

### Windows service
An msi build is planned, but in the meantime you can manually install the app as a windows service by doing the following:

```
c:\windows\system32\sc.exe create TFAgent binPath="C:\path\to\exe\tfagent.exe"
```
The app expects the configuration file to be in either the local directory (with the exe), or in the
%AppData%\TFAgent directory.

To remove the service, use the following command:
```
c:\windows\system32\sc.exe delete WINService
```

If controlling the app via commandline or PowerShell, you can also start/stop/query the app:
```
c:\windows\system32\sc.exe start WINService

c:\windows\system32\sc.exe stop WINService

c:\windows\system32\sc.exe query WINService
```
> [!NOTE]
> Powershell has a 'sc' cmdlet that is run instead of sc.exe if you don't use the full path.

### Config

Sample config file is below.

```
logfile: c:\folder\logs\logfile.log
loglevel: debug
service_heartbeat: true
transfers:
  - name: Result Files 
    source_directory: c:\filesource
    transfertype: sftp
    username: tfuser
    privatekey: c:\path_to_key\id_ed25519_test.priv
    server: 192.168.214.128
    port: 22
    remotepath: incoming
    streaming: false
    overwrite_dest: false
    action_on_success: archive
    action_on_fail: archive
    archive_dest: c:\filesource\archive
    fail_dest: c:\filesource\fail
    filter: "\\.(txt|csv)$"

  - name: Log Files
    source_directory: c:\data\code\testfolder\tf2
    transfertype: sftp
    destination: /path/to/destination2
    username: tfuser
    privatekey: c:\data\code\testfolder\id_ed25519_test.priv
    server: 192.168.214.128
    port: 22
    remotepath: incoming
    streaming: True
    filter: "\\.(jpg|png|gif)$"
```
### Transfer options
| Name | Option | Description |
| --- | --- | --- |
| name | freeform text | This is a human readable label for the transfer |
| transfertype | sftp | In the current version only sftp transfers are supported. Future versions will add scp and local as valid transfer types. |
| username | text | remote username |
| privatekey | file | location of the private key for authenticating with the remote host |
| server | hostname or IP | The remote host to connect and send file |
| port | number | remote port between 1-65535 (default is 22) |
| remotepath | text | The remote path where the file will be sent (required) |
| streaming | true/false | Whether the local file is static or is a streaming file like a log file. This will be used to determine how and when to transfer the file, or file contents. Currently streaming files are not supported. (default: false) |
| overwrite_dest | true/false | Whether to overwrite the file on the destination system if it already exists. (default: false) |
| action_on_success |archive/delete/none | Whether to move the file to an archive directory on successful transfer (default: none) |
| archive_dest | text | The directory to move the file to on success (default: source_directory\archive) |
| action_on_fail |archive/delete/none | Whether to move the file to a fail directory if the transfer fails (default: none) |
| fail_dest | text | The directory to move the file to on fail (default: source_directory\fail) |
| filter | regular expression | a regex string that is used to determine which file(s) to transfer within the source_directory |

