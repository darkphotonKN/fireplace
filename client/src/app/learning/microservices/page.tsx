export default function MicroservicesPage() {
  return (
    <div className="space-y-8">
      <h2 className="mb-4">Microservices Learning Path</h2>
      <p>
        Explore the world of microservices architecture and how to build
        scalable, distributed systems.
      </p>

      <div className="border rounded-lg p-5">
        <h3 className="text-lg mb-3">Topics Covered</h3>
        <ul className="list-disc list-inside space-y-2">
          <li>Service discovery and registry</li>
          <li>API gateways and communication patterns</li>
          <li>Containerization with Docker</li>
          <li>Orchestration with Kubernetes</li>
          <li>Event-driven architecture</li>
        </ul>
      </div>
    </div>
  );
}
