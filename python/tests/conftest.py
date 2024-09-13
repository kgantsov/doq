import asyncio

import httpx
import pytest

from faker import Faker


fake = Faker()


@pytest.fixture(scope="session")
def event_loop():
    """Overrides pytest default function scoped event loop"""
    policy = asyncio.get_event_loop_policy()
    loop = policy.new_event_loop()
    yield loop
    loop.close()


@pytest.fixture(scope="session")
async def fixt_http_client():
    async with httpx.AsyncClient(base_url='http://localhost:8001') as client:
        yield client


@pytest.fixture(scope="package")
async def fixt_generate_queue(fixt_http_client):
    created_names = []

    async def _generate_queue(type: str, name: str):
        # name = f'{"-".join(fake.words(3))}'

        request_data = {
            "type": type,
            "name": name,
        }

        response = await fixt_http_client.post(
            "/API/v1/queues",
            json=request_data
        )
        assert response.status_code == 200, response.text

        resp_data = response.json()

        assert resp_data["status"] == 'CREATED'
        assert resp_data["type"] == type
        assert resp_data["name"] == name

        created_names.append(name)

        return resp_data

    yield _generate_queue

    for name in created_names:
        await fixt_http_client.delete(f"/API/v1/queues/{name}")

@pytest.fixture(scope="package")
async def fixt_fair_queue(fixt_generate_queue):
    group = await fixt_generate_queue(
        type="fair",
        name="fair-transcode"
    )

    yield group

@pytest.fixture(scope="package")
async def fixt_regular_queue(fixt_generate_queue):
    group = await fixt_generate_queue(
        type="delayed",
        name="registration"
    )

    yield group

@pytest.fixture(scope="package")
async def fixt_priority_queue(fixt_generate_queue):
    group = await fixt_generate_queue(
        type="delayed",
        name="login"
    )

    yield group
