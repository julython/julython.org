Julython
========

.. image:: https://api.travis-ci.org/julython/julython.org.png?branch=master
    :target: https://travis-ci.org/julython/julython.org

July is for Programming python 
-----------------------

July 1st to July 31st, 31 days and nights of writing code.

Julython is a month to try something new, something you have had
rolling around in your brain for a while. This could be something
simple like 'build the next Google' or 'replace email'. Or you
could try something hard like 'build a Django website'. All that
matters is that you open source it and write it during the month of
July or January.

Why only 31 days? Why not all year? Well, for one we all have lives
and jobs which get in the way. Having a month set aside where we
all get together and write code will allow us to rely on each other
to keep us on track towards our goals. There will also be a little
bit of healthy competition and public flogging to encourage everyone
to finish.

Goals
------

Since it is very hard to quantify code we have come up with a
simple metric to decide the 'winners' of the competition. The goal
is to commit at least once a day for the entire month. If you are
working on the next twitter or Instagram you don't have to give your
code away. Your commits could be just to a local git or mercurial
repository on your machine. Since there are no real prizes you will
only be cheating yourself by committing 30 days of lorem ipsum.

For those on the leader board though you will have to push your
commits to a public repository which everyone will clearly be able
to see if you're padding your stats.

Help
----

This site is constaintly be tweaked and occasionally problems arise.
If you run into errors or just have general questions hit us up:

Follow us on twitter at `@julython <https://twitter.com/#!/julython>`_.

Email us `help@julython.org <mailto:help@julython.org>`_.

Or join us on freenode `#julython <https://botbot.me/freenode/julython/>`_.


Hacking
-------

The site is written in Django and hosted on Rackspace. To install the dev
environment. Fork this repo and create a virtualenv then install all the
requirements::

    $ pip install -r requirements-dev.txt

Database
++++++++

Create an initial database with some dummy data::

	$ fab install

Run the server and verify you can login with the admin user ``username=admin``
``password=password``::

	$ python manage.py runserver

While the server is running you can test the webhook and create dummy commits
with the command::

	$ fab load:admin@example.com

Media
++++++

We are using `grunt <http://gruntjs.com/>`_ to manage all the assets
so you'll need to have a recent version of
`node and npm installed <http://nodejs.org/>`_ then run::

    $ fab install

Once the modules have been installed you can either `compile` or `watch` the
css/javascript files::

    $ fab compile
    or
    $ fab watch

Testing
+++++++

Run the test suite before submitting any pull requests. You can run
them all like::

    $ fab test

To output coverage report run::

    $ fab test:cover

To run pep8 tests::

    $ fab pep8

Authentication
++++++++++++++

In order to set up Twitter authentication:

#. `Register an application with Twitter <https://dev.twitter.com/apps/new>`_
#. Ensure you have set a Callback URL for your twitter app. This can be
   any valid URL, such as your blog URL. Without this setting your twitter
   app will be locked in 'desktop' mode and you will be unable to
   authenticate.
#. Open ``july/secrets.py`` and add the consumer key and secret provided
   by Twitter for your app.

Translations
------------

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

