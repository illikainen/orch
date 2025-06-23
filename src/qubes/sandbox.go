package qubes

// revive:disable-next-line:function-result-limit
func SandboxPaths() (ro []string, rw []string, dev []string, err error) {
	ro = []string{
		"/var/run/qubes/qrexec-agent",
	}

	dev = []string{
		"/dev/xen/evtchn",
		"/dev/xen/gntalloc",
		"/dev/xen/privcmd",
		"/dev/xen/xenbus",
	}

	return ro, nil, dev, nil
}
