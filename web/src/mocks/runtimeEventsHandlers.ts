/**
 * MSW handlers for runtime events API.
 * Returns mock data for Storybook and tests.
 */
import { http, HttpResponse } from 'msw';
import { generateMockEvents } from '../test-utils/mockRuntimeEvents';

let cachedEvents = generateMockEvents(20);

export const runtimeEventsHandlers = [
  http.get('*/api/v1/runtime-events', ({ request }) => {
    const url = new URL(request.url);
    const limit = Number(url.searchParams.get('limit') ?? 100);
    const workflowId = url.searchParams.get('workflowId');
    const opId = url.searchParams.get('opId');

    let filtered = cachedEvents;
    if (workflowId) {
      filtered = filtered.filter((e) => e.workflowId === workflowId);
    }
    if (opId) {
      filtered = filtered.filter((e) => e.opId === opId);
    }

    // Return as JsonValue[] (serialized protobuf JSON)
    return HttpResponse.json({
      events: filtered.slice(0, limit).map((event) => ({
        id: event.id,
        occurredAt: {
          seconds: String(event.occurredAt?.seconds ?? '0'),
          nanos: event.occurredAt?.nanos ?? 0,
        },
        source: event.source,
        severity: event.severity,
        kind: event.kind,
        message: event.message,
        workflowId: event.workflowId,
        opId: event.opId,
        site: event.site,
        workerId: event.workerId,
        payload: event.payload ?? {},
      })),
    });
  }),
];
