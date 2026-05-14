export default function GenAIPage() {
  return (
    <div className="space-y-8">
      <h2 className="mb-4">Generative AI Learning Path</h2>
      <p>
        Discover the latest techniques in generative AI and how to integrate
        them into your applications.
      </p>

      <div className="border rounded-lg p-5">
        <h3 className="text-lg mb-3">Topics Covered</h3>
        <ul className="list-disc list-inside space-y-2">
          <li>Transformer architectures</li>
          <li>Large Language Models (LLMs)</li>
          <li>Prompt engineering</li>
          <li>Retrieval-Augmented Generation (RAG)</li>
          <li>Fine-tuning and evaluation</li>
        </ul>
      </div>
    </div>
  );
}
