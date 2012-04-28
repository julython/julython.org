
from google.appengine.ext import db

class Section(db.Model):
    """Simple model for handling section content"""
    
    title = db.StringProperty(required=True)
    order = db.IntegerProperty(default=1)
    blurb_one = db.TextProperty()
    blurb_two = db.TextProperty()
    blurb_three = db.TextProperty()
    
    def __unicode__(self):
        return self.title
    