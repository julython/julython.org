from django import forms

class EditUserForm(forms.Form):
    # Match Twitter
    first_name = forms.CharField(max_length=255, required=True)
    last_name = forms.CharField(max_length=255, required=False)
    description = forms.CharField(label="About me", max_length=160, required=False,
        widget=forms.TextInput(attrs={'class': 'span4'}))
    url = forms.CharField(max_length=255, required=False)
    location = forms.CharField(max_length=160, required=False)

    def __init__(self, *args, **kwargs):
        user = kwargs.pop('user', None)
        super(EditUserForm, self).__init__(*args, **kwargs)
        if user:
            self.fields['first_name'].initial=getattr(user, 'first_name', None)
            self.fields['last_name'].initial=getattr(user, 'last_name', None)
            self.fields['description'].initial=getattr(user, 'description', None)
            self.fields['url'].initial=getattr(user, 'url', None)
            self.fields['location'].initial=getattr(user, 'location', None)
            
        
        
class CommitForm(forms.Form):
    
    message = forms.CharField(required=True)
    timestamp = forms.CharField(required=False)
    url = forms.URLField(verify_exists=False, required=False)
    email = forms.EmailField(required=False)
    author = forms.CharField(required=False)
    name = forms.CharField(required=False)
    hash = forms.CharField(required=False)
    
    def clean_timestamp(self):
        data = self.cleaned_data.get('timestamp')
        if data:
            import datetime
            data = datetime.datetime.fromtimestamp(float(data))
        return data

class ProjectForm(forms.Form):
    
    url = forms.URLField(required=True)
    forked = forms.BooleanField(required=False, initial=False)
    parent_url = forms.URLField(required=False)
    
    def clean_parent_url(self):
        data = self.cleaned_data
        if data['parent_url'] == '':
            return None
        return data['parent_url']