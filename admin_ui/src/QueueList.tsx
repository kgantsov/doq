import { Box, Heading } from "@chakra-ui/react";
import QueueTable from "./components/QueueTable";

const QueueList = () => {
  return (
    <Box p={5}>
      <Heading mb={5}>Queues</Heading>
      <QueueTable />
    </Box>
  );
};

export default QueueList;
