import logging

from social_auth.models import UserSocialAuth

from july.people.models import Commit
from july.game.models import Player


def social_auth_user(backend, uid, user=None, *args, **kwargs):
    """Return UserSocialAuth account for backend/uid pair or None if it
    doesn't exists.

    Raise AuthAlreadyAssociated if UserSocialAuth entry belongs to another
    user.
    """
    social_user = UserSocialAuth.get_social_auth(backend.name, uid)
    if social_user:
        if user and social_user.user != user:
            merge_users(user, social_user.user, commit=True)
        elif not user:
            user = social_user.user
    return {'social_user': social_user,
            'user': user,
            'new_association': False}


def merge_users(new_user, old_user, commit=False):
    """
    Merge the users together.

    Args:
      * new_user: User to move items to.
      * old_user: User to move items from.
      * commit: (bool) Actually preform the operations.
    """
    logging.info("Merging %s (%s) into: %s (%s)", old_user, old_user.id,
                 new_user, new_user.id)
    if not commit:
        for o in UserSocialAuth.objects.filter(user=old_user):
            logging.info("Found Auth: %s", o)
        logging.info("Found %s commits",
                     Commit.objects.filter(user=old_user).count())
        logging.info("Found %s projects", old_user.projects.count())
        for p in Player.objects.filter(user=old_user):
            logging.info("Player: %s, %s points: %s", p, p.game, p.points)
        logging.info("Merge player by adding --commit")
    else:
        UserSocialAuth.objects.filter(user=old_user).update(user=new_user)
        for commit in Commit.objects.filter(user=old_user):
            commit.user = new_user
            # Run save individually to trigger the post save hooks.
            commit.save()
        for project in old_user.projects.all():
            new_user.projects.add(project)
        old_user.is_active = False
        old_user.save()
        new_user.save()
        logging.info("Merged")
