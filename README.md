# Julython

## July is for Programming

July 1st to July 31st, 31 days and nights of writing code.

Julython is a month to try something new, something you have had rolling around in your brain for a while. This could be something simple like 'build the next Google' or 'replace email'. Or you could try something hard like 'build a Django website'. All that matters is that you open source it and write it during the month of July or January.

Why only 31 days? Why not all year? Well, for one we all have lives and jobs which get in the way. Having a month set aside where we all get together and write code will allow us to rely on each other to keep us on track towards our goals. There will also be a little bit of healthy competition and public flogging to encourage everyone to finish.

## Goals

Since it is very hard to quantify code we have come up with a simple metric to decide the 'winners' of the competition. The goal is to commit at least once a day for the entire month. If you are working on the next twitter or Instagram you don't have to give your code away. Your commits could be just to a local git or mercurial repository on your machine. Since there are no real prizes you will only be cheating yourself by committing 30 days of lorem ipsum.

For those on the leader board though you will have to push your commits to a public repository which everyone will clearly be able to see if you're padding your stats.

## Help

This site is constaintly be tweaked and occasionally problems arise. If you run into errors or just have general questions hit us up:

Follow us on twitter at `@julython <https://twitter.com/#!/julython>`\_.

Email us `help@julython.org <mailto:help@julython.org>`\_.

## Hacking

This site was originally a Django website (preserved in `june`), we have since converted this to FastAPI and React (`july` and `ui` subfolders). If you want to get started make sure you have `uv` and `docker` installed:

- https://docs.astral.sh/uv/getting-started/installation/
- https://www.docker.com/get-started/

Next if you are on Windows please consider installing Linux as we will not support you ;) All the following commands require `make` which is a rather old technology but still highly useful.

```bash
# To list out the available commands
$ make

# Install requirements
$ make setup

# Run the tests
$ make test

# Run the service locally (this may need the auth and webhook settings listed below)
$ make dev
```

If the tests pass for you great that means you are ready to start hacking. Change some stuff then run the tests again and see that you didn't break anything. If you add a new feature add tests for it and create a pull request.

### Authentication

For local authentication you will need to create a oauth client in github and point it at the callback url:

- http://localhost:8000/auth/callback

Edit the `.env` file that was created in `make setup` and add the client id and secret:

```bash
# project secrets
GITHUB_CLIENT_ID='<your client id>'
GITHUB_CLIENT_SECRET='<your client secret>'
JSON_LOGGING=false
IS_DEV=true
IS_LOCAL=true
DEBUG=true
```

### Webhooks

Github requires that the urls are public and have valid certs. In order to test they recommend using `smee.io`:

https://docs.github.com/en/webhooks/testing-and-troubleshooting-webhooks/testing-webhooks

TLDR: Create a new channle on https://smee.io and then save that in your `.env` file:

```bash
...
GITHUB_WEBHOOK_URL=https://smee.io/<your channel>
```

Restart your local dev server `make dev` and visit http://localhost:8000 once you have authenticated you can view your profile and add a webhook to a repo. Push a commit and verify it gets added correctly.

You can view the database with `adminer` at http://localhost:8080 the host/user/password are all `postgres`. (The database type is also PostgreSQL obviously)

### Adding a Game

Now that you got auth and webhooks working you are going to need a game to score points locally. You can supply any valid date and it will create a game for that period.

```bash
$ export DATE=$(date -I)
$ make game args="--active $DATE"
```

After you have created a game you can verify it is working correctly by pushing a commit to the repo you have connected with a webhook locally.

Congratulations you are now a Julython Hacker!
