// App.tsx
import React from "react";
import {
  BrowserRouter as Router,
  Route,
  Routes,
  useParams,
} from "react-router-dom";
import { Box, Heading } from "@chakra-ui/react";
import QueueList from "./components/QueueList";
import QueueDetails from "./components/QueueDetails";
import ServerList from "./components/ServerList";
import Navbar from "./components/Navbar"; // Import the Navbar component
import { Toaster } from "./components/ui/toaster";

const QueueDetailsRoute = () => {
  const { queueName } = useParams<{ queueName: string }>();

  if (!queueName) return <></>;

  return (
    <Box p={5}>
      <Heading mb={5}>Queue Details</Heading>
      <QueueDetails queueName={queueName} />
    </Box>
  );
};

const App: React.FC = () => {
  return (
    <Router>
      <Navbar /> {/* Add the Navbar here */}
      <Routes>
        <Route path="/" element={<QueueList />} />
        <Route path="/servers" element={<ServerList />} />
        <Route path="/queues/:queueName" element={<QueueDetailsRoute />} />
      </Routes>
      <Toaster />
    </Router>
  );
};

export default App;
