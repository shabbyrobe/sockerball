Sockerball
==========

`sockerball` is/was an experiment I hacked around on in 2018 to search for
solutions to some of the problems and irritants I experienced while working on
a large backend based around a shared TCP+WebSocket protocol for a museum
experience.

It's also full of experiments to try to work out some coherent patterns for
dealing with channel-heavy Go code's tendency to turn into "channel soup".
I think it probably failed in that regard (especially as I let the complexity
spread), but I might have another crack at taming it at a later date.

I ran out of time and puff with it though, so what you see here is where it
ended up when I had to move on to other stuff.

It is named `sockerball` because I was kicking it around a lot in all
directions (and will probably continue to do so at some point).


## Expectation Management

`sockerball` is under-tested, incomplete, experimental and subject to change
at any time without warning.

If you want to use it, please either fork it or vendor it into your repo (don't
forget to retain LICENSE!)

Feel free to submit issues, though I may not be able to attend to them promptly
(or at all). PRs are unlikely to be accepted.

