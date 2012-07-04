Julython
========

July is for Python
------------------

July 1st to July 31st, 31 days and nights of writing Python code. 

Julython is a month to try something new, something you have had
rolling around in your brain for a while. This could be something
simple like 'build the next Google' or 'replace email'. Or you 
could try something hard like 'build a Django website'. All that
matters is that you write it in Python and during the month of
July. 

Why only 31 days? Why not all year. Well, for one we all have lives
and jobs which get in the way. Having a month set aside where we 
all get together and write code will allow us to rely on each other
to keep us on track towards our goals. There will also be a little
bit of healthy competition and public flogging to encourge everyone
to finish.

Goals
-----

Since it is very hard to quantify code we have come up with a 
simple metric to decide the 'winners' of the competition. The goal
is to commit at least once a day for the entire month. If you are 
working on the next twitter or instigram you don't have to give your
code away. Your commits could be just to a local git or mecurial
repository on your machine. Since there is no real prizes you will
only be cheating yourself by commiting 30 days of lorem ipsum.

For those on the leader board though you will have to push your
commits to a public repository which everyone will clearly be able
to see if your padding your stats.

Site
----

The site is a work in progress right now and volunteers are welcome
also we could use some nice free hosting (**hint hint**). In the
mean time follow us on the twitter ``@julython``.


Hacking
-------

The site is written in Django and hosted on Appengine. You can use this
gist to setup your dev environment to help out::

    https://gist.github.com/2839803

In order to set up Twitter authentication:

1. `Register an application with Twitter <https://dev.twitter.com/apps/new>`_
2. ``cp july/secrets.py.template july/secrets.py``
3. Open ``july/secrets.py`` and add the consumer key and secret provided
   by Twitter for your app.

In order to maintain internationalization (i18n) support, please try
to make sure and run the following command after changing any translated text:

    $ django-admin.py makemessages

If you can, edit the accompanying message files (you'll find them in
``july/locale`` with an extension of ``.po``), then run:

    $ django-admin.py compilemessages

Ping the following developers in your pull request or commit message
if you'd like to have new strings translated:

- ``locale/ja``: `modocache <https://github.com/modocache>`_
- ``locale/ro``: @florinm
