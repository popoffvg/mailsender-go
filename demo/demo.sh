#!/bin/bash
export ADDR="0.0.0.0:8084"
export mail="popoffvg@gmail.com"

function title {
    echo
    echo "--------"
    echo $1
}

mailing_greeting=$(cat mailing_greeting.json | sed "s/test@test.net/$mail/g")
mailing_interview=$(cat mailing_interview.json | sed "s/test@test.net/$mail/g")

title CREATE
curl -H 'Content-Type: application/json' \
    -d "$mailing_greeting" "$ADDR/mailing"

title LIST
curl -H 'Content-Type: application/json' \
    -d "$mailing_interview" "$ADDR/mailing"

curl "$ADDR/mailing"
