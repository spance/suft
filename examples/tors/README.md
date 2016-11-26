# Usage

1. for server

```
./tors -l :<Port> -b <Bandwidth>
```

2. for client - pull file from remote

```
rsync -P --blocking-io -e './tors -r <Remote-Host>:<Remote-Port>' x:<Remote-File> .
```

3. for client - push file to remote

```
rsync -P --blocking-io -e './tors -r <Remote-Host>:<Remote-Port> -b <Bandwidth>' <Local-File> x:
```
