// App.tsx
import React from "react";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { ChakraProvider } from "@chakra-ui/react";
import QueueList from "./QueueList";
import QueueDetails from "./QueueDetails";
import Navbar from "./components/Navbar"; // Import the Navbar component

const App: React.FC = () => {
  return (
    <ChakraProvider>
      <Router>
        <Navbar /> {/* Add the Navbar here */}
        <Routes>
          <Route path="/" element={<QueueList />} />
          <Route path="/queues/:queueName" element={<QueueDetails />} />
        </Routes>
      </Router>
    </ChakraProvider>
  );
};

export default App;
