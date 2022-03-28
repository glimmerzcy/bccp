OLD_CGO_ENABLED=$(go env CGO_ENABLED)
OLD_GOOS=$(go env GOOS)
OLD_GOARCH=$(go env GOARCH)

go env -w CGO_ENABLED=0
go env -w GOOS=linux
go env -w GOARCH=amd64
go build main.go

go env -w CGO_ENABLED="${OLD_CGO_ENABLED}"
go env -w GOOS="${OLD_GOOS}"
go env -w GOARCH="${OLD_GOARCH}"

servers=(106.3.97.70 106.3.97.36 106.3.97.28 106.3.97.67 106.3.97.45 106.3.97.120 106.3.97.212)

# shellcheck disable=SC2088
path="~/consensus/server"
file="$path/main"

for server in "${servers[@]}"
do
  host="root@${server}"
  echo "mkdir -p $path" | ssh "$host"
  scp main "$host:$file"
  echo "chmod +x $file && cd $path && nohup $file &" | ssh "$host"
done
