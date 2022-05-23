## config-watcher
A small utility watching for changes in a configuration folder. Once a change is detected the utility sends SIGTERM signal to a process.
The goal is to facilitate a configuration reload for processes such as [fluentbit - issue 365](https://github.com/fluent/fluent-bit/issues/365) that doesn't provide such functionality, yet. The utility runs as a sidecar in a pod, shareing the process namespace with the target process. It uses [/proc/[pid]/cmdline](https://man7.org/linux/man-pages/man5/proc.5.html) to identify the target to which the os signal is sent.
