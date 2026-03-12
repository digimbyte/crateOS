if [ -n "${CRATEOS_SHELL_BYPASS:-}" ]; then
    return 0
fi

if [ -n "${SSH_CONNECTION:-}" ] || [ -n "${SSH_TTY:-}" ]; then
    return 0
fi

if [ ! -x /usr/local/bin/crateos-shell-wrapper ]; then
    return 0
fi

tty_path="$(tty 2>/dev/null || true)"
if [ "${tty_path}" != "/dev/tty1" ]; then
    return 0
fi

if [ "${SHLVL:-1}" != "1" ]; then
    return 0
fi

export CRATEOS_SHELL_BYPASS=1
exec /usr/local/bin/crateos-shell-wrapper
