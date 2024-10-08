import { Box, Heading } from "@chakra-ui/react";
import QueueDetails from "./components/QueueDetails";
import { useParams } from "react-router-dom";

const QueueDetailsPage = () => {
  const { queueName } = useParams<{ queueName: string }>();

  if (!queueName) return <></>;

  return (
    <Box p={5}>
      <Heading mb={5}>Queue Details</Heading>
      <QueueDetails queueName={queueName} />
    </Box>
  );
};

export default QueueDetailsPage;
