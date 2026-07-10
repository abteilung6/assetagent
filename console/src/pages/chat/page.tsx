import type React from "react";

import { Composer } from "@/components/chat/composer";
import { MessageList } from "@/components/chat/message-list";
import { useChat } from "@/hooks/use-chat";

const ChatPage: React.FC = () => {
  const { messages, send, isPending, error } = useChat();

  return (
    <div className="-m-4 flex min-h-0 flex-1 flex-col overflow-hidden">
      <MessageList messages={messages} isPending={isPending} />
      {error ? (
        <p className="shrink-0 px-4 pb-2 text-sm text-destructive">
          Failed to send message. Check that the API and Ollama are running.
        </p>
      ) : null}
      <Composer onSend={send} disabled={isPending} />
    </div>
  );
};

export default ChatPage;
