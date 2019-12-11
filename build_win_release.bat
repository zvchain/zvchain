@set basedir=%~dp0
set basedir=%basedir:\=/%
git submodule sync
git submodule update --init
sh.exe %basedir%tvm/ctvm/buildlib.sh
copy .\network\p2p\p2p_api.h .\network /y
copy .\network\p2p\windows\libp2pcore.a .\network /y
copy .\tvm\ctvm\py\tvm.h .\tvm /y
copy .\tvm\ctvm\examples\zvm\libtvm.a .\tvm /y
go clean -cache %basedir%cmd/gzv
go build -tags release -o ./bin/gzv.exe %basedir%cmd/gzv