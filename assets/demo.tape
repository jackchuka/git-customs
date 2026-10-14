# Regenerate with: vhs assets/demo.tape (from the repository root)
# Requires gitleaks on PATH. Everything runs in /tmp/demo with a throwaway HOME;
# the "gitlab.com" remote is a local bare repo reached through a fake ssh command.

Output assets/demo.gif

Set Shell "bash"
Set FontSize 22
Set Width 1440
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type `export D=/tmp/demo && rm -rf "$D" && mkdir -p "$D/bin" "$D/home" && go build -o "$D/bin/git-customs" .`
Enter
Type `unset XDG_CONFIG_HOME GIT_DIR GIT_WORK_TREE; export HOME="$D/home" GIT_CONFIG_GLOBAL="$D/home/.gitconfig" GIT_CONFIG_NOSYSTEM=1 PATH="$D/bin:$PATH" GIT_SSH_COMMAND=fake-ssh GIT_SSH_VARIANT=simple`
Enter
Type `printf '#!/bin/sh\ncd /tmp/demo/srv && exec sh -c "$2"\n' > "$D/bin/fake-ssh" && chmod +x "$D/bin/fake-ssh" && git init -q --bare "$D/srv/alice/weather-cli.git"`
Enter
Type `git config --global user.name Alice && git config --global user.email alice@example.com && git config --global init.defaultBranch main`
Enter
Type `mkdir -p ~/.config/git-customs && printf 'commands = ["gitleaks stdin --no-banner --redact -v"]\n\n[repos]\npublic  = true\nprivate = false\n' > ~/.config/git-customs/config.toml`
Enter
Type `git init -q "$D/weather-cli" && cd "$D/weather-cli" && git remote add origin git@gitlab.com:alice/weather-cli.git && printf 'region = "us-east-1"\naws_access_key_id = "AKIAQ4DEMOK3YZW7XN2P"\n' > config.toml && git add . && git commit -qm "Add AWS config"`
Enter
Type `export PS1='\[\e[1;35m\]$ \[\e[0m\]' && clear`
Enter
Sleep 4s
Show

Type "git customs install"
Sleep 400ms
Enter
Sleep 2s

Type "cat ~/.config/git-customs/config.toml"
Sleep 400ms
Enter
Sleep 3s

Type "clear"
Enter
Type "cat config.toml && git push origin main"
Sleep 400ms
Enter
Sleep 5s

Type "clear"
Enter
Type `echo 'region = "us-east-1"' > config.toml && git commit -qa --amend --no-edit`
Sleep 400ms
Enter
Sleep 1s
Type "git push origin main"
Sleep 400ms
Enter
Sleep 4s
