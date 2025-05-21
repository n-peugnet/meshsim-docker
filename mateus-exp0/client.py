import asyncio

import os
from nio import AsyncClient, ErrorResponse, MatrixRoom, Response, RoomMessageText, RoomVisibility

SERVER_NAME = os.environ['SYNAPSE_SERVER_NAME']

async def message_callback(room: MatrixRoom, event: RoomMessageText) -> None:
    print(
        f"Message received in room {room.display_name}\n"
        f"{room.user_name(event.sender)} | {event.body}",
        flush=True
    )


async def main() -> None:
    client = AsyncClient(f"http://{SERVER_NAME}:8008", f"@matthew:{SERVER_NAME}")
    client.add_event_callback(message_callback, RoomMessageText)

    print(await client.login(password="secret"), flush=True)
    # "Logged in as @matthew:SERVER_NAME device id: RANDOMDID"

    response: Response
    if SERVER_NAME == "node0":
        response = await client.room_create(
            visibility=RoomVisibility.public,
            alias="test",
        )
    else:
        response = await client.join("#test:node0")

    if isinstance(response, ErrorResponse):
        raise Exception(response.message)

    await client.room_send(
        room_id=response.room_id,
        message_type="m.room.message",
        content={"msgtype": "m.text", "body": "Hello world!"},
    )
    await client.sync_forever(timeout=30000)  # milliseconds

asyncio.run(main())
