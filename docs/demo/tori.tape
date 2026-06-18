# Demo tape for tori. Rendered with ascii-gif (github.com/tamnd/ascii-gif),
# which supplies the window chrome and theme; this file is just the action.
#
#   ascii-gif render docs/demo/tori.tape -o docs/static/demo.gif
#
# tori must be on PATH inside the recording shell.

Hide
Type "export PS1='$ ' PATH=/tmp/toribin:$PATH && cd $(mktemp -d) && clear"
Enter
Show

Sleep 600ms
Type "tori archive 20 -o ."
Sleep 700ms
Enter
Sleep 4s

Type "tori info x/status-20"
Sleep 700ms
Enter
Sleep 3.5s

Type "ls x/status-20"
Sleep 700ms
Enter
Sleep 3s

Type "tori serve x/status-20"
Sleep 700ms
Enter
Sleep 3s
Ctrl+C
Sleep 1.2s
