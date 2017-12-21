# sku 

sku context

sku context "set context"

## install

```
brew install glide
```

set the GOPATH in e.g. your ~/.zshrc

```
export GOPATH=/Users/USERNAME/src/go
```

restart your console and run

now clone or move this repo into your GOPATH e.g. `~/src/go/src/github.com/sandstorm` and change to this direcotory. Than run: 

```
glide update --strip-vendor
./build.sh
```

To run this command everywhere create a symlink
```
ln -s /Users/USERNAME/src/go/src/github.com/sandstorm/sku/sku /usr/local/bin/
```
