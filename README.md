## config-watcher
A small utility watching for changes in a configuration folder. Once a change is detected the utility sends SIGTERM signal to a process.
The goal is to facilitate a configuration reload for processes such as [fluentbit - issue 365](https://github.com/fluent/fluent-bit/issues/365)
that don't provide such functionality, yet. The utility runs as a main container in a pod restarting the child process when changes are detected.
At the moment the utility is mainly used to handle fluent-bit restarts. It uses official [flient-bit](https://hub.docker.com/r/fluent/fluent-bit) images
