FROM python:2.7.11-wheezy

COPY requirements.txt ./requirements.txt
COPY requirements-dev.txt ./requirements-dev.txt

RUN apt-get install -y libmysqlclient-dev libssl-dev
RUN pip install -r requirements.txt

WORKDIR /usr/local/julython.org
