# CONXEC a simple tool to run commands in a distroless container

When you need to run commands in a container you use `docker run` or `docker exec` and you have pass the command and the arguments. But it dosn't work if you want to run a command in a distroless or a slim container as they lack on basic linux shell and commands binaries.

This tool is a simple solution to this problem. It will run a command in a distroless container even if it doesn't have a shell.

This project is heavily inspired by [iximiuz/cdebug](https://github.com/iximiuz/cdebug)
and it almost did exactly the same thing. But there are some imrovments:
- In cdebug to work you have to have the target container running as root. In conxec you can run it as a nonroot user which is a very common practice. It will help you to debug your application in a production environment.
- It allows you to install and use aditional static applications available on the wolfi. It is very useful if you want to debug your application with some extra tools like `strace` or `tcpdump`.
- It also allows you to mount a local directory inside the container. It is very useful if you need some local files or apps to debug your application.

## Installation
go install github.com/debasishbsws/conxec@latest

## Documentation

