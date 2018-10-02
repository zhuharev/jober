go build -o jober cmd/jober/*.go
upx jober
rsync -avzrP --files-from=conf/files.txt ./ god@simplecloud:/home/god/sites/jober
ssh god@simplecloud "sudo stop jober && sudo start jober"
