import datetime
from datetime import date
from django.conf import settings

oneday = datetime.timedelta(days=1)
ABUSE_DELTA = datetime.timedelta(days=settings.ABUSE_LIMIT)


def to_date(raw):
    return datetime.datetime.strptime(raw, '%x')


def to_string(date_time):
    return date_time.strftime('%x')


class AbuseMiddleware(object):

    def _can_report_abuse(self, request):
        def can_report_abuse():
            reset_date = to_string(date.today() - ABUSE_DELTA)
            abuse_date = request.session.get('abuse_date', reset_date)
            return to_date(abuse_date).date() < date.today()
        return can_report_abuse

    def _abuse_reported(self, request):
        def abuse_reported():
            if not request.can_report_abuse():
                return False
            reset_date = to_string(date.today() - ABUSE_DELTA)
            abuse_date = to_date(request.session.get('abuse_date', reset_date))
            if abuse_date.date() + ABUSE_DELTA < date.today():
                abuse_date = date.today() - ABUSE_DELTA
            abuse_date += oneday
            request.session['abuse_date'] = to_string(abuse_date)
            return True
        return abuse_reported

    def process_request(self, request):
        request.can_report_abuse = self._can_report_abuse(request)
        request.abuse_reported = self._abuse_reported(request)
