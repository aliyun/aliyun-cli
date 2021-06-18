# Aliyun CLI Toolkit

## Why not choose a popular library

Because aliyun cli need process unknown flags, and the following popular library did not support this feature
- https://github.com/spf13/cobra
- https://github.com/urfave/cli

## Code Structure

### Command

.Run

.Help

### Flag

### FlagSet

### Ctx

### i18n

### TODO

- [x] Suggestions
- [x] Support shorthand flag, -a -b
- [x] Flag alias
- [x] Auto complete framework
- [ ] Help document generation (ref: https://github.com/spf13/cobra#generating-documentation-for-your-command)
- [ ] Optimize --help message view
- [ ] Support `help --waiter`
- [x] Support Group Options
- [x] Optimize noticeable error
- [ ] (low)Support shorthand combination -ab
