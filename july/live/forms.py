
from django import forms
from django.template.defaultfilters import escape

class MessageForm(forms.Form):
    
    username = forms.CharField(required=True)
    picture_url = forms.URLField(required=True)
    message = forms.CharField(required=True)
    commit_hash = forms.CharField(required=False)
    url = forms.URLField(required=False)
    project = forms.URLField(required=False)
    
    def clean_username(self):
        data = self.cleaned_data.get('username')
        return escape(data)
    
    def clean_message(self):
        data = self.cleaned_data.get('message')
        return escape(data)
    
    def clean_commit_hash(self):
        data = self.cleaned_data.get('commit_hash')
        return escape(data)