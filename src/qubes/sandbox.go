package qubes

// revive:disable-next-line:function-result-limit
func SandboxPaths() (ro []string, rw []string, dev []string, err error) {
	ro = []string{
		"/etc/qubes-release",
		"/var/run/qubes",
		"/var/run/qubes/qrexec-agent",
		"/var/run/qubesd.sock",
	}

	dev = []string{
		"/dev/xen/evtchn",
		"/dev/xen/gntalloc",
		"/dev/xen/privcmd",
		"/dev/xen/xenbus",
	}

	return ro, rw, dev, nil
}
