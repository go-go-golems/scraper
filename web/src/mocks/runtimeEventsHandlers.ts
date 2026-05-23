/**
 * Runtime events are delivered through the sessionstream websocket endpoint.
 * Component stories should pass mock RuntimeEventV1 values directly instead of
 * mocking the removed REST/SSE runtime-events API.
 */
export const runtimeEventsHandlers = [];
