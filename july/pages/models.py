
from django.db import models

class Section(models.Model):
    """Simple model for handling section content"""

    title = models.CharField(max_length=200)
    order = models.IntegerField(default=1)
    # the slug is the CSS class name, too
    slug = models.CharField(max_length=200)

    # this is pretty darn hacky. :)
    use_blurb_one = models.BooleanField()
    blurb_one_image_url = models.CharField(max_length=200)
    blurb_one_title = models.CharField(max_length=200)
    blurb_one_content = models.TextField()

    use_blurb_two = models.BooleanField()
    blurb_two_image_url = models.CharField(max_length=200)
    blurb_two_title = models.CharField(max_length=200)
    blurb_two_content = models.TextField()

    use_blurb_three = models.BooleanField()
    blurb_three_image_url = models.CharField(max_length=200)
    blurb_three_title = models.CharField(max_length=200)
    blurb_three_content = models.TextField()

    @property
    def blurbs(self):
        return _Blurbs(self)

    @property
    def blocks(self):
        all_blurbs = self.blurbs
        if len(all_blurbs):
            blurbs = all_blurbs[:]
            last_block = None
            if blurbs[0].image_url:
                blurb = blurbs.pop(0)
                yield _Block(blurb)
            if blurbs[-1].image_url:
                blurb = blurbs.pop(-1)
                last_block = _Block(blurb)
            yield _Block(
                *blurbs, title=self.title, slug=self.slug)
            if last_block:
                yield last_block

    def __unicode__(self):
        return self.title

_BLURB_IDS = ["one", "two", "three"]
_BOOTSTRAP_COLUMNS = 12
_COLUMN_MULTIPLIER = _BOOTSTRAP_COLUMNS / len(_BLURB_IDS)

class _Block(object):
    """A Content block. Hackerific."""

    def __init__(self, *blurbs, **kwargs):
        self.title = kwargs.get("title")
        self.slug = kwargs.get("slug")
        self.blurbs = blurbs

    # lol...look how amazing Django templates are at keeping
    # template and business logic separate!
    @property
    def bootstrap_width(self):
        return len(self.blurbs) * _COLUMN_MULTIPLIER

class _Blurbs(object):
    """Iterator for blurbs."""

    def __init__(self, section):
        self._section = section
        self._blurb_index = -1
        self._blurbs = []
        self._populate_blurbs()

    def _populate_blurbs(self):
        """Cache the Blurb instances."""
        for blurb_id in _BLURB_IDS:
            if getattr(self._section, "use_blurb_%s" % blurb_id):
                self._blurbs.append(_Blurb(self._section, blurb_id))

    def __len__(self):
        return len(self._blurbs)

    def __getitem__(self, key):
        return self._blurbs.__getitem__(key)

class _Blurb(object):
    """Simple access for a blurb."""

    def __init__(self, section, blurb_id):
        self._section = section
        self._blurb_id = blurb_id

    def _get_attribute_value(self, key):
        attr_name = "blurb_%s_%s" % (self._blurb_id, key)
        return getattr(self._section, attr_name)

    @property
    def title(self):
        return self._get_attribute_value("title")

    @property
    def content(self):
        raw_content = self._get_attribute_value("content")
        # we should parse this as markdown now...
        return raw_content

    @property
    def image_url(self):
        return self._get_attribute_value("image_url")
