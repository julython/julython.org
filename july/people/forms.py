from django import forms
from django.utils.translation import ugettext_lazy
from models import Location, Team

from july.utils import check_location


class EditAddressForm(forms.Form):

    def __init__(self, *args, **kwargs):
        self.user = kwargs.pop('user', None)
        super(EditAddressForm, self).__init__(*args, **kwargs)

        if self.user:
            self.fields['address_line1'].initial = getattr(
                self.user, 'address_line1', None)
            self.fields['address_line2'].initial = getattr(
                self.user, 'address_line2', None)
            self.fields['city'].initial = getattr(
                self.user, 'city', None)
            self.fields['state'].initial = getattr(
                self.user, 'state', None)
            self.fields['country'].initial = getattr(
                self.user, 'country', None)
            self.fields['postal_code'].initial = getattr(
                self.user, 'postal_code', None)

    address_line1 = forms.CharField(
        label=ugettext_lazy('Address'),
        max_length=255, required=True
    )
    address_line2 = forms.CharField(
        label=ugettext_lazy('Address Line 2'),
        max_length=255, required=False
    )
    city = forms.CharField(
        label=ugettext_lazy('City'),
        max_length=20, required=True
    )
    state = forms.CharField(
        label=ugettext_lazy('State / Region'),
        max_length=12, required=True
    )
    country = forms.CharField(
        label=ugettext_lazy('Country'),
        max_length=25, required=True
    )
    postal_code = forms.CharField(
        label=ugettext_lazy('Postal Code'),
        max_length=12, required=True
    )


class EditUserForm(forms.Form):
    # Match Twitter
    first_name = forms.CharField(
        label=ugettext_lazy('First name'),
        max_length=255, required=True
    )
    last_name = forms.CharField(
        label=ugettext_lazy('Last name'),
        max_length=255, required=False
    )
    description = forms.CharField(
        label=ugettext_lazy("About me"), max_length=160, required=False,
        widget=forms.TextInput(attrs={'class': 'span6'})
    )
    url = forms.CharField(
        label=ugettext_lazy('URL'),
        max_length=255, required=False,
        widget=forms.TextInput(attrs={'class': 'span4'})
    )

    location = forms.CharField(
        label=ugettext_lazy('Location'),
        help_text=ugettext_lazy(
            'Note new locations need to be approved first before'
            ' they will show up.'),
        max_length=160, required=False,
        widget=forms.TextInput(
            attrs={
                'data-bind': 'typeahead: $data.filterLocation'
            }
        )
    )

    team = forms.CharField(
        label=ugettext_lazy('Team'),
        help_text=ugettext_lazy(
            'Note new teams need to be approved first before'
            ' they will show up'),
        max_length=160, required=False,
        widget=forms.TextInput(
            attrs={
                'data-bind': 'typeahead: $data.filterTeam'
            }
        )
    )

    gittip = forms.CharField(
        label=ugettext_lazy("Gittip Username"), required=False)

    email = forms.EmailField(
        label=ugettext_lazy("Add Email Address"), required=False)

    def __init__(self, *args, **kwargs):
        self.user = kwargs.pop('user', None)
        self._gittip = None
        super(EditUserForm, self).__init__(*args, **kwargs)
        if self.user:
            self.fields['first_name'].initial = self.user.first_name
            self.fields['last_name'].initial = self.user.last_name
            self.fields['description'].initial = self.user.description
            self.fields['url'].initial = self.user.url
            if self.user.location:
                self.fields['location'].initial = self.user.location.name
            if self.user.team:
                self.fields['team'].initial = self.user.team.name
            # initialize the emails
            self.emails = set(self.user.social_auth.filter(provider="email"))
            self._gittip = self.user.get_provider("gittip")
            if self._gittip:
                self.fields['gittip'].initial = self._gittip.uid

    def clean_location(self):
        location = self.data.get('location', '')
        if not location:
            return
        location = check_location(location)
        if not location:
            error_msg = ugettext_lazy(
                "Specified location is invalid"
            )
            raise forms.ValidationError(error_msg)
        return Location.create(location)

    def clean_team(self):
        team = self.data.get('team', '')
        return Team.create(team)

    def clean_gittip(self):
        uid = self.cleaned_data['gittip']
        if not uid:
            if self._gittip is not None:
                self._gittip.delete()
            return None
        if self._gittip is not None:
            self._gittip.uid = uid
            self._gittip.save()
            return uid
        else:
            try:
                self.user.add_auth_id('gittip:%s' % uid)
            except:
                error_msg = ugettext_lazy(
                    "This gittip username is already in use, if this is not"
                    " right please email help@julython.org"
                )
                raise forms.ValidationError(error_msg)
        return uid

    def clean_email(self):
        email = self.cleaned_data['email']
        if not email:
            return None
        if email in [auth.uid for auth in self.emails]:
            error_msg = ugettext_lazy("You already have that email address!")
            raise forms.ValidationError(error_msg)

        # add the email address to the user, this will cause a ndb.put()
        try:
            self.user.add_auth_email(email)
        except Exception:
            error_msg = ugettext_lazy(
                "This email is already taken, if this is not right please "
                "email help@julython.org "
            )
            raise forms.ValidationError(error_msg)

        # Defer a task to fix orphan commits
        # TODO - make this a celery task?
        # deferred.defer(fix_orphans, email=email)
        return email


class CommitForm(forms.Form):

    message = forms.CharField(required=True)
    timestamp = forms.CharField(required=False)
    url = forms.URLField(required=False)
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
