# Changes

- Build all binaries with CGO_ENABLED=1 explicitly set. This will allow the `patchelf` tool to correctly rewrite the path to the interpreter when using an alternative glibc.
