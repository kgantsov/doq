// App.tsx
import React from "react";
import { BrowserRouter as Router, Route, Routes } from "react-router-dom";
import { Provider } from "./components/ui/provider";
import QueueList from "./QueueList";
import QueueDetails from "./QueueDetails";
import Navbar from "./components/Navbar"; // Import the Navbar component
import { Toaster } from "./components/ui/toaster";

const App: React.FC = () => {
  return (
    <Provider>
      <Router>
        <Navbar /> {/* Add the Navbar here */}
        <Routes>
          <Route path="/" element={<QueueList />} />
          <Route path="/queues/:queueName" element={<QueueDetails />} />
        </Routes>
        <Toaster />
      </Router>
    </Provider>
  );
};

export default App;
