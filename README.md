# Fake Compiler

Pretend as if something is compiling or busying in your terminal.

<img src="assets/image.png" alt="alt text" style="zoom: 33%;" />


## Usage

### Run compiler: `run` subcommand

Run over a directory: `fake-compiler run -d path_to_compile -C compiler_type ` 
  - Currently, only `cxx` compiler_type is implemented, which looks like cmake compiling logs
  - `fake-compiler` will iterate through the whole `path_to_compile` directory and print compiling logs of all files with `.cpp/.c/.S` extension

Or run with a config file: `fake-compiler run -c config_file`
  - The config file contains parsed result of some directory. It has specific format, you should generate it by `gen` subcommand 
  - Actually it is equivalent to `-d` option, except that `fake-compiler` now no longer needs to explicitly iterate through and parse the directory everytime

Optional flag: `-t threads`: specify the number of threads, default: 16
  - since `fake-compiler` does not actually do the compiling stuff, this flag essentially specifies how many threads are sleeping at the same time

### Generate config file: `gen` subcommand
You can generate config file by `fake-compiler gen -C compiler_type -d path_to_compile -o output_file`
  - The generated file is bound to specified compiler_type



This repository is shipped with an example config file, placed at `examples/linux-6.12.17_cxx`, generated over linux 6.12.17 source code with `cxx` compiler_type. Try it out (need Go environment): `go run . run -c examples/linux-6.12.17_cxx`



## Build

1. Setup Go environment, minimal version required: 1.22
2. Run `make build` 