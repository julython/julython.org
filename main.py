from flask import Flask, render_template
from july import settings

app = Flask('julython')
app.config.from_object(settings)

@app.route('/')
def index():
    from july.pages.models import Section
    
    sections = Section.all().fetch(100)
    return render_template('index.html', sections=sections, user=None)