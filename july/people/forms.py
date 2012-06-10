from django import forms

class EditUserForm(forms.Form):
    about_me = forms.CharField(widget=forms.Textarea, required=False)
    url = forms.CharField(max_length=255, required=False)
    facebook_url = forms.CharField(max_length=255, required=False)
    email = forms.EmailField(max_length=255)

    def __init__(self, *args, **kwargs):
        user = kwargs.pop('user', None)
        super(EditUserForm, self).__init__(*args, **kwargs)
        if user:
            self.fields['about_me'].initial=getattr(user, 'about_me', None)
            self.fields['url'].initial=getattr(user, 'url', None)
            self.fields['facebook_url'].initial=getattr(user, 'facebook_url', None)
            self.fields['email'].initial=user.email
        
        
class CommitForm(forms.Form):
    
    message = forms.CharField(required=True)
    timestamp = forms.CharField(required=False)
    url = forms.URLField(verify_exists=False, required=False)
    email = forms.EmailField(required=False)
    author = forms.CharField(required=False)
    name = forms.CharField(required=False)
    hash = forms.CharField(required=False)

class ProjectForm(forms.Form):
    
    url = forms.URLField(required=True)
    forked = forms.BooleanField(required=False)
    parent_url = forms.URLField(required=False)