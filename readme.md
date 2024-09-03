# Cosmos Signer

This application can sign potentially arbitrary off-line generated transactions
from any Cosmos-SDK based blockchain. It is itself technically a Cosmos-SDK
application, but it is not meant to be run as a blockchain node, rather as
a standalone application. All the node-related code has been stripped out or
the functionality has been disabled.

The core component is the `x/signer` module, which is canonically a Cosmos-SDK
module, however it just provides the CLI commands to sign transactions in the
`x/signer/client/cli` package.
The CLI code extends and-reuses the `x/auth/client/cli`
package from the Cosmos-SDK, which already provides code to sign transactions.
The only purpose of the `x/signer` module is to allow dynamic registration
of the proto types required to encode and decode the provided transactions.

Since the `x/signer` module is a Cosmos-SDK module, it is also possible to
add its functionality to any other compatible Cosmos-SDK application.
However, the utility of such module might be questionable on a specific chain,
where generic-purpose signers are generally not needed.

## How

The core functionality is available in the `x/signer/client/cli` package.
Other important modifications for the CLI itself are available at
`cmd/cosmos-signer/cmd`, in particular in `commands.go`.

To allow dynamic registration of the proto types, the `x/signer` module
uses [go plugins](https://pkg.go.dev/plugin) to load the required types
from per-chain plugins.

These plugins are built - as of now - by ripping out and modifying the original
chain source code, of which we just need the `types` packages of the used modules,
and replacing the `go.mod` with the same `go.mod` as the comos-signer (plus
adaptations, which suggest this process might not be easily automatable).
The plugin `main.go` will have to look something like this:

```go
package main

import (
    govtypes "github.com/atomone-hub/govgen/x/gov/types"
)

var Govgen_gov_v1beta1_RegisterLegacyAminoCodec = govtypes.RegisterLegacyAminoCodec
var Govgen_gov_v1beta1_RegisterInterfaces = govtypes.RegisterInterfaces

func main() {}
```

Examples can be found for `gaia` and `govgen` in the `plugins` directory.

In both cases, we used - or adapted the code to support - Cosmos-SDK v0.50.x,
and synchronized any other dependency with the root application.
This was done as an intentional excercise to test compatibility across different
Cosmos-SDK versions, and to check complexity and feasibility of getting to a
truly generic signer, which should somehow be "version-agnostic".

The challenge is that go plugins - although seem very appropriate for this
task - need to be built with the exact same dependencies as the root application,
which is not necessarily easy to achieve in the context of Cosmos-SDK apps.

## Build

To build the signer, run:

```bash
./build.sh
```

which will build the signer and the plugins for `gaia` and `govgen`, and
make them available in the `build` directory.

## Run

The application should be familiar to anyone who has used the Cosmos-SDK CLI.
Three commands are available for now. First is the `keys` commands for key management.
Second is `tx sign`. Finally is `tx verify-signatures`. The interface is the same as the
`x/auth`-provided commands,but `--offline` flag is mandatory, as obviously the signer
is meant to be used to sign offline-generated transactions. There are also a few
notable additions of flags that are available with the cosmos signer:

- `--bech32-prefix`: to specify the bech32 prefix that we need to use for
  encoding the addresses. It defaults to `cosmos`, and can be used throghout
  the whole application (e.g. even for `keys` commands).
- `--prefix-pub`: to specify the prefix for the public key. It defaults to
  `pub`, and can be used throghout the whole application (e.g. even for
  `keys` commands).

and specifically for the `tx sign` command:

- `--plugins-dir`: to specify the directory where the plugins are located.
   It is a mandatory flag.
