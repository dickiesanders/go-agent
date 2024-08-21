## Ignore warnings

The warnings you are seeing are related to deprecated system APIs being used by `gopsutil` when compiled on macOS. Unfortunately, these are just warnings about deprecated APIs and don’t typically affect the functionality of your program.

### Options for Suppressing These Warnings

#### 1. **Ignoring the Warnings Globally**

One way to suppress the warnings globally is by using the `CGO_CFLAGS` environment variable to pass specific flags to the C compiler. You can pass `-Wno-deprecated-declarations` to tell the compiler to suppress warnings about deprecated declarations.

Run the following command to suppress the warnings:

```bash
CGO_CFLAGS="-Wno-deprecated-declarations" go run cmd/main.go
```

This sets the `CGO_CFLAGS` variable to suppress the specific warning related to deprecated declarations.

#### 2. **Ignoring Warnings During Build**

If you're building a binary and want to suppress these warnings during the build process, you can pass the `CGO_CFLAGS` directly to the build command:

```bash
CGO_CFLAGS="-Wno-deprecated-declarations" go build -o myagent cmd/main.go
```

This will build your Go program without showing the deprecated warnings.

### Important Considerations

- **Warnings Are Not Errors**: These warnings won't break your program, so while it's good to be aware of them, they don't necessarily need to be suppressed unless they are cluttering your output or causing concern.
- **Keep an Eye on Dependencies**: Since these warnings relate to deprecated APIs in macOS, it’s a good idea to monitor the `gopsutil` library for updates that might address the deprecation and eventually remove these warnings.

By using the `CGO_CFLAGS` environment variable as shown above, you can suppress the deprecated warnings and prevent them from being printed during build or run.