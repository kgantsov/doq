import {
  Link,
  Center,
  Spinner,
  Flex,
  Spacer,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Button,
  Box,
  Heading,
  useDisclosure,
} from "@chakra-ui/react";
import { ChevronDownIcon } from "@chakra-ui/icons";
import { Badge } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import { Queue } from "../types/queues";
import { getQueues } from "../api/queues";
import { useQuery } from "@tanstack/react-query";
import CreateQueueModal from "./CreateQueueModal";
import { createColumnHelper } from "@tanstack/react-table";
import { DataTable } from "./DataTable";

const columnHelper = createColumnHelper<Queue>();

const columns = [
  columnHelper.accessor("name", {
    cell: (info) => {
      const name = info.getValue();
      return (
        <Link as={RouterLink} to={`/queues/${name}`}>
          {name}
        </Link>
      );
    },
    header: "Name",
  }),
  columnHelper.accessor("type", {
    cell: (info) => {
      const type = info.getValue();
      return (
        <Badge colorScheme={type === "delayed" ? "teal" : "teal"}>{type}</Badge>
      );
    },
    header: "Type",
  }),
  columnHelper.accessor("ready", {
    cell: (info) => info.getValue(),
    header: "ready",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("unacked", {
    cell: (info) => info.getValue(),
    header: "unacked",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("total", {
    cell: (info) => info.getValue(),
    header: "total",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("enqueue_rps", {
    cell: (info) => info.getValue(),
    header: "enqueue_rps",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("dequeue_rps", {
    cell: (info) => info.getValue(),
    header: "dequeue_rps",
    meta: {
      isNumeric: true,
    },
  }),
  columnHelper.accessor("ack_rps", {
    cell: (info) => info.getValue(),
    header: "ack_rps",
    meta: {
      isNumeric: true,
    },
  }),
];

const QueueList = () => {
  const { isOpen, onOpen, onClose } = useDisclosure();

  const { isPending, data } = useQuery({
    queryKey: ["queues"],
    queryFn: getQueues,
    refetchInterval: 1000,
  });

  if (isPending) {
    return (
      <Center h="90vh" color="white">
        <Spinner
          thickness="4px"
          speed="0.65s"
          emptyColor="gray.200"
          color="blue.500"
          size="xl"
        />
      </Center>
    );
  }

  const queues: Queue[] = data?.queues || [];

  return (
    <>
      <Box p={5}>
        <Flex>
          <Heading mb={5}>Queues</Heading>
          <Spacer />
          <Menu>
            <MenuButton as={Button} rightIcon={<ChevronDownIcon />}>
              Actions
            </MenuButton>
            <MenuList>
              <MenuItem onClick={onOpen}>Create Queue</MenuItem>
            </MenuList>
          </Menu>
        </Flex>
        <DataTable columns={columns} data={queues} />
      </Box>
      <CreateQueueModal isOpen={isOpen} onClose={onClose} />
    </>
  );
};

export default QueueList;
