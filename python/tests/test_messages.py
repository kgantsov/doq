import asyncio
import json
import random
import time
import pytest

from faker import Faker

fake = Faker()


@pytest.mark.asyncio
async def test_fair_priority_queue(fixt_http_client, fixt_fair_queue):
    tests = [
        {
            "name": "test-1",
            "messages": [
                {
                    "content": '{"id": 1, "name": "Clifford Gordon"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 2, "name": "Cynthia Thomas"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 3, "name": "Joseph Smith"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 4, "name": "Lisa Anderson"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 5, "name": "Tyler Norris"}',
                    "priority": 10,
                    "group": "customer-2"
                },
                {
                    "content": '{"id": 6, "name": "Derek Pennington"}',
                    "priority": 10,
                    "group": "customer-2"
                },
                {
                    "content": '{"id": 7, "name": "Allison Richardson"}',
                    "priority": 10,
                    "group": "customer-3"
                },
                {
                    "content": '{"id": 8, "name": "Sharon Madden"}',
                    "priority": 10,
                    "group": "customer-4"
                }
            ],
            "expected_message_indexes": [0, 4, 6, 7, 1, 5, 2, 3]
        }
    ]

    queue_name = fixt_fair_queue['name']

    for test in tests:
        for message in test['messages']:
            response = await fixt_http_client.post(
                url=f"/API/v1/queues/{queue_name}/messages",
                json=message
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'ENQUEUED'
            assert data["priority"] == message["priority"]
            assert data["content"] == message["content"]

        for index in test['expected_message_indexes']:
            response = await fixt_http_client.get(
                url=f"/API/v1/queues/{queue_name}/messages?ack=true"
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'DEQUEUED'
            assert data["content"] == test['messages'][index]["content"]
            assert data["priority"] == test['messages'][index]["priority"]
            assert data["id"] is not None


@pytest.mark.asyncio
async def test_regular_queue(fixt_http_client, fixt_regular_queue):
    tests = [
        {
            "name": "test-1",
            "messages": [
                {
                    "content": '{"id": 1, "name": "Clifford Gordon"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 2, "name": "Cynthia Thomas"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 3, "name": "Joseph Smith"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 4, "name": "Lisa Anderson"}',
                    "priority": 10,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 5, "name": "Tyler Norris"}',
                    "priority": 10,
                    "group": "customer-2"
                },
                {
                    "content": '{"id": 6, "name": "Derek Pennington"}',
                    "priority": 10,
                    "group": "customer-2"
                },
                {
                    "content": '{"id": 7, "name": "Allison Richardson"}',
                    "priority": 10,
                    "group": "customer-3"
                },
                {
                    "content": '{"id": 8, "name": "Sharon Madden"}',
                    "priority": 10,
                    "group": "customer-4"
                }
            ],
            "expected_message_indexes": [0, 1, 2, 3, 4, 5, 6, 7]
        }
    ]

    queue_name = fixt_regular_queue['name']

    for test in tests:
        for message in test['messages']:
            response = await fixt_http_client.post(
                url=f"/API/v1/queues/{queue_name}/messages",
                json=message
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'ENQUEUED'
            assert data["priority"] == message["priority"]
            assert data["content"] == message["content"]

        for index in test['expected_message_indexes']:
            response = await fixt_http_client.get(
                url=f"/API/v1/queues/{queue_name}/messages?ack=true"
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'DEQUEUED'
            assert data["content"] == test['messages'][index]["content"]
            assert data["priority"] == test['messages'][index]["priority"]
            assert data["id"] is not None


@pytest.mark.asyncio
async def test_priority_queue(fixt_http_client, fixt_priority_queue):
    tests = [
        {
            "name": "test-1",
            "messages": [
                {
                    "content": '{"id": 1, "name": "Clifford Gordon"}',
                    "priority": 123,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 2, "name": "Cynthia Thomas"}',
                    "priority": 523,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 3, "name": "Joseph Smith"}',
                    "priority": 35,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 4, "name": "Lisa Anderson"}',
                    "priority": 7863,
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 5, "name": "Tyler Norris"}',
                    "priority": 12,
                    "group": "customer-2"
                },
                {
                    "content": '{"id": 6, "name": "Derek Pennington"}',
                    "priority": 4,
                    "group": "customer-2"
                },
                {
                    "content": '{"id": 7, "name": "Allison Richardson"}',
                    "priority": 1,
                    "group": "customer-3"
                },
                {
                    "content": '{"id": 8, "name": "Sharon Madden"}',
                    "priority": 4555,
                    "group": "customer-4"
                }
            ],
            "expected_message_indexes": [6, 5, 4, 2, 0, 1, 7, 3]
        }
    ]

    queue_name = fixt_priority_queue['name']

    for test in tests:
        for message in test['messages']:
            response = await fixt_http_client.post(
                url=f"/API/v1/queues/{queue_name}/messages",
                json=message
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'ENQUEUED'
            assert data["priority"] == message["priority"]
            assert data["content"] == message["content"]

        for index in test['expected_message_indexes']:
            response = await fixt_http_client.get(
                url=f"/API/v1/queues/{queue_name}/messages?ack=true"
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'DEQUEUED'
            assert data["content"] == test['messages'][index]["content"]
            assert data["priority"] == test['messages'][index]["priority"]
            assert data["id"] is not None

@pytest.mark.slow
@pytest.mark.asyncio
async def test_delayed_queue(fixt_http_client, fixt_regular_queue):
    tests = [
        {
            "name": "test-1",
            "messages": [
                {
                    "content": '{"id": 1, "name": "Clifford Gordon"}',
                    "priority": int(time.time() + 2),
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 2, "name": "Cynthia Thomas"}',
                    "priority": int(time.time() + 1),
                    "group": "customer-1"
                },
                {
                    "content": '{"id": 3, "name": "Joseph Smith"}',
                    "priority": 10,
                    "group": "customer-1"
                },
            ],
            "expected_message_indexes": [2, 1, 0]
        }
    ]

    queue_name = fixt_regular_queue['name']

    for test in tests:
        for message in test['messages']:
            response = await fixt_http_client.post(
                url=f"/API/v1/queues/{queue_name}/messages",
                json=message
            )
            assert response.status_code == 200, response.text

            data = response.json()

            assert data["status"] == 'ENQUEUED'
            assert data["priority"] == message["priority"]
            assert data["content"] == message["content"]

        response = await fixt_http_client.get(
            url=f"/API/v1/queues/{queue_name}/messages?ack=true"
        )
        assert response.status_code == 200, response.text

        data = response.json()

        assert data["status"] == 'DEQUEUED'
        assert data["content"] == test['messages'][2]["content"]
        assert data["priority"] == test['messages'][2]["priority"]
        assert data["id"] is not None

        response = await fixt_http_client.get(
            url=f"/API/v1/queues/{queue_name}/messages?ack=true"
        )
        assert response.status_code == 400, response.text

        await asyncio.sleep(1.1)

        response = await fixt_http_client.get(
            url=f"/API/v1/queues/{queue_name}/messages?ack=true"
        )
        assert response.status_code == 200, response.text

        data = response.json()

        assert data["status"] == 'DEQUEUED'
        assert data["content"] == test['messages'][1]["content"]
        assert data["priority"] == test['messages'][1]["priority"]
        assert data["id"] is not None

        response = await fixt_http_client.get(
            url=f"/API/v1/queues/{queue_name}/messages?ack=true"
        )
        assert response.status_code == 400, response.text

        await asyncio.sleep(1.1)

        response = await fixt_http_client.get(
            url=f"/API/v1/queues/{queue_name}/messages?ack=true"
        )
        assert response.status_code == 200, response.text

        data = response.json()

        assert data["status"] == 'DEQUEUED'
        assert data["content"] == test['messages'][0]["content"]
        assert data["priority"] == test['messages'][0]["priority"]
        assert data["id"] is not None
