# Manual time tracker

It helps me focus on the right things, and it also reminds me to take a break.

It's a command-line tool with [rofi](https://github.com/davatorium/rofi) integration. It uses just one simple SQLite database with one table `log`.


## How do I use it?
I run in the background
```
timefor daemon
```

I have key-bindings for
```sh
# specify an activity name using rofi
timefor select

# finish the current activity
timefor finish

# reject the current activity
timefor reject
```

I integrate it into my [status bar](https://github.com/vivien/i3blocks) using
```
timefor show --i3blocks
```

So I always see my current activity on the screen. If `timefor` activity is work-related, then I should work. If I want to surf the internet for fun, then I should switch `timefor` activity to `@surf` or similar.

Daemon will send notification using `notify-send` after 80 minutes by default, when I see such notification I plan to
move away from my laptop in the near time.

There is no report command yet, but I can get reports from SQLite directly
```sh
# execute sqlite3 with db file
timefor db

# in SQLite session, today's activities grouped by name
sqlite>
SELECT name, SUM(duration) / 60 duration_minutes
FROM log
WHERE date(started, 'unixepoch', 'localtime') = date()
GROUP BY name;
```

P.S. It's a simplified version of a similar GTK based tool [tider](https://github.com/naspeh/tider).