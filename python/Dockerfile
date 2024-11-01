FROM python:3.12

ENV INSTALL_DIR=/opt/doq/e2d
RUN mkdir -p $INSTALL_DIR

WORKDIR $INSTALL_DIR

# RUN pip install ipython requests

RUN pip install poetry==1.2.0 setuptools==75.3.0

ADD requirements.txt .

RUN pip install -r requirements.txt

ADD . $INSTALL_DIR


# CMD [ "sleep", "1" ]