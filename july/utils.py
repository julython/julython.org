# coding: utf-8
import requests

def check_location(location):
    resp = requests.get('http://maps.googleapis.com/maps/api/geocode/json',
        params={'address': location, 'sensor': 'false'})
    resp.raise_for_status()

    data = resp.json()
    if data['status'] == 'ZERO_RESULTS':
        return False
    try:
        return data['results'][0]['formatted_address']
    except (KeyError, IndexError):
        return False


